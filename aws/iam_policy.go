package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"github.com/hashicorp/go-multierror"
	"sync"
	"time"
)

// Returns the ARN of all customer managed policies
func getAllLocalIamPolicies(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := iam.New(session)

	var allIamPolicies []*string

	err := svc.ListPoliciesPages(
		&iam.ListPoliciesInput{Scope: aws.String(iam.PolicyScopeTypeLocal)},
		func(page *iam.ListPoliciesOutput, lastPage bool) bool {
			for _, policy := range page.Policies {
				if shouldIncludeIamPolicy(policy, excludeAfter, configObj) {
					allIamPolicies = append(allIamPolicies, policy.Arn)
				}
			}
			return !lastPage
		},
	)

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return allIamPolicies, nil
}

// Delete all iam customer managed policies. Caller is responsible for pagination (no more than 100/request)
func nukeAllIamPolicies(session *session.Session, policyArns []*string) error {
	svc := iam.New(session)

	if len(policyArns) == 0 {
		logging.Logger.Debug("No IAM Policies to nuke")
	}

	//Probably not required since pagination is handled by the caller
	if len(policyArns) > 100 {
		logging.Logger.Errorf("Nuking too many IAM Policies at once (100): Halting to avoid rate limits")
		return TooManyIamPolicyErr{}
	}

	//No Bulk Delete exists, do it with goroutines
	logging.Logger.Debug("Deleting all IAM Policies")
	wg := new(sync.WaitGroup)
	wg.Add(len(policyArns))
	errChans := make([]chan error, len(policyArns))
	for i, arn := range policyArns {
		errChans[i] = make(chan error, 1)
		go deleteIamPolicyAsync(wg, errChans[i], svc, arn)
	}
	wg.Wait()

	//Collapse the errors down to one
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Logger.Errorf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking IAM Policy",
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

// Removes an IAM Policy from AWS, designed to run as a goroutine
func deleteIamPolicyAsync(wg *sync.WaitGroup, errChan chan error, svc *iam.IAM, policyArn *string) {
	defer wg.Done()
	var multierr *multierror.Error

	//Detach any entities the policy is attached to
	err := detachPolicyEntities(svc, policyArn)
	if err != nil {
		multierr = multierror.Append(multierr, err)
	}

	//Get Old Policy Versions
	var versionsToRemove []*string
	err = svc.ListPolicyVersionsPages(&iam.ListPolicyVersionsInput{PolicyArn: policyArn},
		func(page *iam.ListPolicyVersionsOutput, lastPage bool) bool {
			for _, policyVersion := range page.Versions {
				if !*policyVersion.IsDefaultVersion {
					versionsToRemove = append(versionsToRemove, policyVersion.VersionId)
				}
			}
			return !lastPage
		})
	if err != nil {
		multierr = multierror.Append(multierr, err)
	}

	//Delete old policy versions
	for _, versionId := range versionsToRemove {
		_, err = svc.DeletePolicyVersion(&iam.DeletePolicyVersionInput{VersionId: versionId, PolicyArn: policyArn})
		if err != nil {
			multierr = multierror.Append(multierr, err)
		}
	}
	//Delete the policy
	_, err = svc.DeletePolicy(&iam.DeletePolicyInput{PolicyArn: policyArn})
	if err != nil {
		multierr = multierror.Append(multierr, err)
	} else {
		logging.Logger.Debugf("[OK] IAM Policy %s was deleted in global", aws.StringValue(policyArn))
	}

	e := report.Entry{
		Identifier:   aws.StringValue(policyArn),
		ResourceType: "IAM Policy",
		Error:        multierr.ErrorOrNil(),
	}
	report.Record(e)

	errChan <- multierr.ErrorOrNil()
}

func detachPolicyEntities(svc *iam.IAM, policyArn *string) error {
	var allPolicyGroups []*string
	var allPolicyRoles []*string
	var allPolicyUsers []*string
	err := svc.ListEntitiesForPolicyPages(&iam.ListEntitiesForPolicyInput{PolicyArn: policyArn},
		func(page *iam.ListEntitiesForPolicyOutput, lastPage bool) bool {
			for _, group := range page.PolicyGroups {
				allPolicyGroups = append(allPolicyGroups, group.GroupName)
			}
			for _, role := range page.PolicyRoles {
				allPolicyRoles = append(allPolicyRoles, role.RoleName)
			}
			for _, user := range page.PolicyUsers {
				allPolicyUsers = append(allPolicyUsers, user.UserName)
			}
			return !lastPage
		},
	)
	if err != nil {
		return err
	}
	//Detach policy from any users
	for _, userName := range allPolicyUsers {
		detachUserInput := &iam.DetachUserPolicyInput{
			UserName:  userName,
			PolicyArn: policyArn,
		}
		_, err = svc.DetachUserPolicy(detachUserInput)
		if err != nil {
			return err
		}
	}
	//Detach policy from any groups
	for _, groupName := range allPolicyGroups {
		detachGroupInput := &iam.DetachGroupPolicyInput{
			GroupName: groupName,
			PolicyArn: policyArn,
		}
		_, err = svc.DetachGroupPolicy(detachGroupInput)
		if err != nil {
			return err
		}
	}
	//Detach policy from any roles
	for _, roleName := range allPolicyRoles {
		detachRoleInput := &iam.DetachRolePolicyInput{
			RoleName:  roleName,
			PolicyArn: policyArn,
		}
		_, err = svc.DetachRolePolicy(detachRoleInput)
		if err != nil {
			return err
		}
	}
	return err
}

func shouldIncludeIamPolicy(iamPolicy *iam.Policy, excludeAfter time.Time, configObj config.Config) bool {
	if iamPolicy == nil {
		return false
	}

	if excludeAfter.Before(aws.TimeValue(iamPolicy.CreateDate)) {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(iamPolicy.PolicyName),
		configObj.IAMPolicy.IncludeRule.NamesRegExp,
		configObj.IAMPolicy.ExcludeRule.NamesRegExp,
	)
}

// TooManyIamPolicyErr Custom Errors
type TooManyIamPolicyErr struct{}

func (err TooManyIamPolicyErr) Error() string {
	return "Too many IAM Groups requested at once"
}
