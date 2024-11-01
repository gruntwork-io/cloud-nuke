package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/hashicorp/go-multierror"
)

func (ig *IAMGroups) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var allIamGroups []*string
	paginator := iam.NewListGroupsPaginator(ig.Client, &iam.ListGroupsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, iamGroup := range page.Groups {
			if configObj.IAMGroups.ShouldInclude(config.ResourceValue{
				Time: iamGroup.CreateDate,
				Name: iamGroup.GroupName,
			}) {
				allIamGroups = append(allIamGroups, iamGroup.GroupName)
			}
		}
	}

	return allIamGroups, nil
}

// nukeAll - delete all IAM groups.  Caller is responsible for pagination (no more than 100/request)
func (ig *IAMGroups) nukeAll(groupNames []*string) error {
	if len(groupNames) == 0 {
		logging.Debug("No IAM Groups to nuke")
		return nil
	}

	// Probably not required since pagination is handled by the caller
	if len(groupNames) > 100 {
		logging.Errorf("Nuking too many IAM Groups at once (100): Halting to avoid rate limits")
		return TooManyIamGroupErr{}
	}

	// No bulk delete exists, do it with goroutines
	logging.Debug("Deleting all IAM Groups")
	wg := new(sync.WaitGroup)
	wg.Add(len(groupNames))
	errChans := make([]chan error, len(groupNames))
	for i, groupName := range groupNames {
		errChans[i] = make(chan error, 1)
		go ig.deleteAsync(wg, errChans[i], groupName)
	}
	wg.Wait()

	// Collapse the errors down to one
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Errorf("[Failed] %s", err)
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}

	return nil
}

// deleteIamGroup - removes an IAM group from AWS, designed to run as a goroutine
func (ig *IAMGroups) deleteAsync(wg *sync.WaitGroup, errChan chan error, groupName *string) {
	defer wg.Done()
	var multierr *multierror.Error

	// Remove any users from the group
	getGroupInput := &iam.GetGroupInput{
		GroupName: groupName,
	}
	grp, err := ig.Client.GetGroup(ig.Context, getGroupInput)
	for _, user := range grp.Users {
		unlinkUserInput := &iam.RemoveUserFromGroupInput{
			UserName:  user.UserName,
			GroupName: groupName,
		}
		_, err := ig.Client.RemoveUserFromGroup(ig.Context, unlinkUserInput)
		if err != nil {
			multierr = multierror.Append(multierr, err)
		}
	}

	// Detach any policies on the group
	var allPolicies []*string
	paginator := iam.NewListAttachedGroupPoliciesPaginator(ig.Client, &iam.ListAttachedGroupPoliciesInput{GroupName: groupName})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ig.Context)
		if err != nil {
			multierr = multierror.Append(multierr, err)
			break
		}

		for _, iamPolicy := range page.AttachedPolicies {
			allPolicies = append(allPolicies, iamPolicy.PolicyArn)
		}
	}

	for _, policy := range allPolicies {
		unlinkPolicyInput := &iam.DetachGroupPolicyInput{
			GroupName: groupName,
			PolicyArn: policy,
		}
		_, err = ig.Client.DetachGroupPolicy(ig.Context, unlinkPolicyInput)
	}

	// Detach any inline policies on the group
	var allInlinePolicyNames []*string

	policiesPaginator := iam.NewListGroupPoliciesPaginator(ig.Client, &iam.ListGroupPoliciesInput{GroupName: groupName})
	for policiesPaginator.HasMorePages() {
		page, err := policiesPaginator.NextPage(ig.Context)
		if err != nil {
			multierr = multierror.Append(multierr, err)
			break
		}

		for _, policyName := range page.PolicyNames {
			allInlinePolicyNames = append(allInlinePolicyNames, aws.String(policyName))
		}
	}

	for _, policyName := range allInlinePolicyNames {
		_, err = ig.Client.DeleteGroupPolicy(ig.Context, &iam.DeleteGroupPolicyInput{
			GroupName:  groupName,
			PolicyName: policyName,
		})
	}

	// Delete the group
	_, err = ig.Client.DeleteGroup(ig.Context, &iam.DeleteGroupInput{
		GroupName: groupName,
	})
	if err != nil {
		multierr = multierror.Append(multierr, err)
	} else {
		logging.Debugf("[OK] IAM Group %s was deleted in global", aws.ToString(groupName))
	}

	e := report.Entry{
		Identifier:   aws.ToString(groupName),
		ResourceType: "IAM Group",
		Error:        multierr.ErrorOrNil(),
	}
	report.Record(e)

	errChan <- multierr.ErrorOrNil()
}

// TooManyIamGroupErr Custom Errors
type TooManyIamGroupErr struct{}

func (err TooManyIamGroupErr) Error() string {
	return "Too many IAM Groups requested at once"
}
