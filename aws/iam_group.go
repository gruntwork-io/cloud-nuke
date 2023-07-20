package aws

import (
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/hashicorp/go-multierror"
)

func getAllIamGroups(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := iam.New(session)

	var allIamGroups []*string
	err := svc.ListGroupsPages(
		&iam.ListGroupsInput{},
		func(page *iam.ListGroupsOutput, lastPage bool) bool {
			for _, iamGroup := range page.Groups {
				if shouldIncludeIamGroup(iamGroup, excludeAfter, configObj) {
					allIamGroups = append(allIamGroups, iamGroup.GroupName)
				}
			}
			return !lastPage
		},
	)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	return allIamGroups, nil
}

// nukeAllIamGroups - delete all IAM groups.  Caller is responsible for pagination (no more than 100/request)
func nukeAllIamGroups(session *session.Session, groupNames []*string) error {
	svc := iam.New(session)

	if len(groupNames) == 0 {
		logging.Logger.Debug("No IAM Groups to nuke")
		return nil
	}

	//Probably not required since pagination is handled by the caller
	if len(groupNames) > 100 {
		logging.Logger.Errorf("Nuking too many IAM Groups at once (100): Halting to avoid rate limits")
		return TooManyIamGroupErr{}
	}

	//No bulk delete exists, do it with goroutines
	logging.Logger.Debug("Deleting all IAM Groups")
	wg := new(sync.WaitGroup)
	wg.Add(len(groupNames))
	errChans := make([]chan error, len(groupNames))
	for i, groupName := range groupNames {
		errChans[i] = make(chan error, 1)
		go deleteIamGroupAsync(wg, errChans[i], svc, groupName)
	}
	wg.Wait()

	//Collapse the errors down to one
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Logger.Errorf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking IAM Group",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}

	return nil
}

// deleteIamGroup - removes an IAM group from AWS, designed to run as a goroutine
func deleteIamGroupAsync(wg *sync.WaitGroup, errChan chan error, svc *iam.IAM, groupName *string) {
	defer wg.Done()
	var multierr *multierror.Error

	//Remove any users from the group
	getGroupInput := &iam.GetGroupInput{
		GroupName: groupName,
	}
	grp, err := svc.GetGroup(getGroupInput)
	for _, user := range grp.Users {
		unlinkUserInput := &iam.RemoveUserFromGroupInput{
			UserName:  user.UserName,
			GroupName: groupName,
		}
		_, err := svc.RemoveUserFromGroup(unlinkUserInput)
		if err != nil {
			multierr = multierror.Append(multierr, err)
		}
	}

	//Detach any policies on the group
	allPolicies := []*string{}
	err = svc.ListAttachedGroupPoliciesPages(&iam.ListAttachedGroupPoliciesInput{GroupName: groupName},
		func(page *iam.ListAttachedGroupPoliciesOutput, lastPage bool) bool {
			for _, iamPolicy := range page.AttachedPolicies {
				allPolicies = append(allPolicies, iamPolicy.PolicyArn)
			}
			return !lastPage
		},
	)

	for _, policy := range allPolicies {
		unlinkPolicyInput := &iam.DetachGroupPolicyInput{
			GroupName: groupName,
			PolicyArn: policy,
		}
		_, err = svc.DetachGroupPolicy(unlinkPolicyInput)
	}

	// Detach any inline policies on the group
	allInlinePolicyNames := []*string{}
	err = svc.ListGroupPoliciesPages(&iam.ListGroupPoliciesInput{GroupName: groupName},
		func(page *iam.ListGroupPoliciesOutput, lastPage bool) bool {
			logging.Logger.Info("ListGroupPolicies response page: ", page)
			for _, policyName := range page.PolicyNames {
				allInlinePolicyNames = append(allInlinePolicyNames, policyName)
			}
			return !lastPage
		},
	)

	logging.Logger.Info("inline policies: ", allInlinePolicyNames)
	for _, policyName := range allInlinePolicyNames {
		_, err = svc.DeleteGroupPolicy(&iam.DeleteGroupPolicyInput{
			GroupName:  groupName,
			PolicyName: policyName,
		})
	}

	//Delete the group
	_, err = svc.DeleteGroup(&iam.DeleteGroupInput{
		GroupName: groupName,
	})
	if err != nil {
		multierr = multierror.Append(multierr, err)
	} else {
		logging.Logger.Debugf("[OK] IAM Group %s was deleted in global", aws.StringValue(groupName))
	}

	e := report.Entry{
		Identifier:   aws.StringValue(groupName),
		ResourceType: "IAM Group",
		Error:        multierr.ErrorOrNil(),
	}
	report.Record(e)

	errChan <- multierr.ErrorOrNil()
}

// check if iam group should be included based on config rules (RegExp and Exclude After)
func shouldIncludeIamGroup(iamGroup *iam.Group, excludeAfter time.Time, configObj config.Config) bool {
	if iamGroup == nil {
		return false
	}

	if excludeAfter.Before(aws.TimeValue(iamGroup.CreateDate)) {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(iamGroup.GroupName),
		configObj.IAMGroup.IncludeRule.NamesRegExp,
		configObj.IAMGroup.ExcludeRule.NamesRegExp,
	)
}

// TooManyIamGroupErr Custom Errors
type TooManyIamGroupErr struct{}

func (err TooManyIamGroupErr) Error() string {
	return "Too many IAM Groups requested at once"
}
