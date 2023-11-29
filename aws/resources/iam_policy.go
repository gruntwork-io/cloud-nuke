package resources

import (
	"context"
	"sync"

	"github.com/andrewderr/cloud-nuke-a1/config"
	"github.com/andrewderr/cloud-nuke-a1/logging"
	"github.com/andrewderr/cloud-nuke-a1/report"
	"github.com/andrewderr/cloud-nuke-a1/telemetry"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"github.com/hashicorp/go-multierror"
)

// Returns the ARN of all customer managed policies
func (ip *IAMPolicies) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var allIamPolicies []*string

	err := ip.Client.ListPoliciesPages(
		&iam.ListPoliciesInput{Scope: aws.String(iam.PolicyScopeTypeLocal)},
		func(page *iam.ListPoliciesOutput, lastPage bool) bool {
			for _, policy := range page.Policies {
				if configObj.IAMPolicies.ShouldInclude(config.ResourceValue{
					Name: policy.PolicyName,
					Time: policy.CreateDate,
				}) {
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
func (ip *IAMPolicies) nukeAll(policyArns []*string) error {
	if len(policyArns) == 0 {
		logging.Debug("No IAM Policies to nuke")
	}

	//Probably not required since pagination is handled by the caller
	if len(policyArns) > 100 {
		logging.Errorf("Nuking too many IAM Policies at once (100): Halting to avoid rate limits")
		return TooManyIamPolicyErr{}
	}

	//No Bulk Delete exists, do it with goroutines
	logging.Debug("Deleting all IAM Policies")
	wg := new(sync.WaitGroup)
	wg.Add(len(policyArns))
	errChans := make([]chan error, len(policyArns))
	for i, arn := range policyArns {
		errChans[i] = make(chan error, 1)
		go ip.deleteIamPolicyAsync(wg, errChans[i], arn)
	}
	wg.Wait()

	//Collapse the errors down to one
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Errorf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking IAM Policy",
			}, map[string]interface{}{})
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}

	return nil
}

// Removes an IAM Policy from AWS, designed to run as a goroutine
func (ip *IAMPolicies) deleteIamPolicyAsync(wg *sync.WaitGroup, errChan chan error, policyArn *string) {
	defer wg.Done()
	var multierr *multierror.Error

	//Detach any entities the policy is attached to
	err := ip.detachPolicyEntities(policyArn)
	if err != nil {
		multierr = multierror.Append(multierr, err)
	}

	//Get Old Policy Versions
	var versionsToRemove []*string
	err = ip.Client.ListPolicyVersionsPages(&iam.ListPolicyVersionsInput{PolicyArn: policyArn},
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
		_, err = ip.Client.DeletePolicyVersion(&iam.DeletePolicyVersionInput{VersionId: versionId, PolicyArn: policyArn})
		if err != nil {
			multierr = multierror.Append(multierr, err)
		}
	}
	//Delete the policy
	_, err = ip.Client.DeletePolicy(&iam.DeletePolicyInput{PolicyArn: policyArn})
	if err != nil {
		multierr = multierror.Append(multierr, err)
	} else {
		logging.Debugf("[OK] IAM Policy %s was deleted in global", aws.StringValue(policyArn))
	}

	e := report.Entry{
		Identifier:   aws.StringValue(policyArn),
		ResourceType: "IAM Policy",
		Error:        multierr.ErrorOrNil(),
	}
	report.Record(e)

	errChan <- multierr.ErrorOrNil()
}

func (ip *IAMPolicies) detachPolicyEntities(policyArn *string) error {
	var allPolicyGroups []*string
	var allPolicyRoles []*string
	var allPolicyUsers []*string
	err := ip.Client.ListEntitiesForPolicyPages(&iam.ListEntitiesForPolicyInput{PolicyArn: policyArn},
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
		_, err = ip.Client.DetachUserPolicy(detachUserInput)
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
		_, err = ip.Client.DetachGroupPolicy(detachGroupInput)
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
		_, err = ip.Client.DetachRolePolicy(detachRoleInput)
		if err != nil {
			return err
		}
	}
	return err
}

// TooManyIamPolicyErr Custom Errors
type TooManyIamPolicyErr struct{}

func (err TooManyIamPolicyErr) Error() string {
	return "Too many IAM Groups requested at once"
}
