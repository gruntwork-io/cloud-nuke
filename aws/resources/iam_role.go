package resources

import (
	"context"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// List all IAM Roles in the AWS account
func (ir *IAMRoles) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var allIAMRoles []*string
	paginator := iam.NewListRolesPaginator(ir.Client, &iam.ListRolesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, iamRole := range page.Roles {
			if ir.shouldInclude(&iamRole, configObj) {
				allIAMRoles = append(allIAMRoles, iamRole.RoleName)
			}
		}
	}

	return allIAMRoles, nil
}

func (ir *IAMRoles) deleteManagedRolePolicies(roleName *string) error {
	policiesOutput, err := ir.Client.ListAttachedRolePolicies(ir.Context, &iam.ListAttachedRolePoliciesInput{
		RoleName: roleName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	for _, attachedPolicy := range policiesOutput.AttachedPolicies {
		arn := attachedPolicy.PolicyArn
		_, err = ir.Client.DetachRolePolicy(ir.Context, &iam.DetachRolePolicyInput{
			PolicyArn: arn,
			RoleName:  roleName,
		})
		if err != nil {
			logging.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
		logging.Debugf("Detached Policy %s from Role %s", aws.ToString(arn), aws.ToString(roleName))
	}

	return nil
}

func (ir *IAMRoles) deleteInlineRolePolicies(roleName *string) error {
	policyOutput, err := ir.Client.ListRolePolicies(ir.Context, &iam.ListRolePoliciesInput{
		RoleName: roleName,
	})
	if err != nil {
		logging.Debugf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	for _, policyName := range policyOutput.PolicyNames {
		_, err := ir.Client.DeleteRolePolicy(ir.Context, &iam.DeleteRolePolicyInput{
			PolicyName: aws.String(policyName),
			RoleName:   roleName,
		})
		if err != nil {
			logging.Debugf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
		logging.Debugf("Deleted Inline Policy %s from Role %s", policyName, aws.ToString(roleName))
	}

	return nil
}

func (ir *IAMRoles) deleteInstanceProfilesFromRole(roleName *string) error {
	profilesOutput, err := ir.Client.ListInstanceProfilesForRole(ir.Context, &iam.ListInstanceProfilesForRoleInput{
		RoleName: roleName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	for _, profile := range profilesOutput.InstanceProfiles {

		// Role needs to be removed from instance profile before it can be deleted
		_, err := ir.Client.RemoveRoleFromInstanceProfile(ir.Context, &iam.RemoveRoleFromInstanceProfileInput{
			InstanceProfileName: profile.InstanceProfileName,
			RoleName:            roleName,
		})
		if err != nil {
			logging.Debugf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		} else {
			_, err := ir.Client.DeleteInstanceProfile(ir.Context, &iam.DeleteInstanceProfileInput{
				InstanceProfileName: profile.InstanceProfileName,
			})
			if err != nil {
				logging.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
		logging.Debugf("Detached and Deleted InstanceProfile %s from Role %s", aws.ToString(profile.InstanceProfileName), aws.ToString(roleName))
	}
	return nil
}

func (ir *IAMRoles) deleteIamRole(roleName *string) error {
	_, err := ir.Client.DeleteRole(ir.Context, &iam.DeleteRoleInput{
		RoleName: roleName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// Delete all IAM Roles
func (ir *IAMRoles) nukeAll(roleNames []*string) error {
	if len(roleNames) == 0 {
		logging.Debug("No IAM Roles to nuke")
		return nil
	}

	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on IAMRoles.MaxBatchSize, however we add a guard here to warn users when the batching fails and has a
	// chance of throttling AWS. Since we concurrently make one call for each identifier, we pick 100 for the limit here
	// because many APIs in AWS have a limit of 100 requests per second.
	if len(roleNames) > 100 {
		logging.Debugf("Nuking too many IAM Roles at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyIamRoleErr{}
	}

	// There is no bulk delete IAM Roles API, so we delete the batch of IAM roles concurrently using go routines
	logging.Debugf("Deleting all IAM Roles")
	wg := new(sync.WaitGroup)
	wg.Add(len(roleNames))
	errChans := make([]chan error, len(roleNames))
	for i, roleName := range roleNames {
		errChans[i] = make(chan error, 1)
		go ir.deleteIamRoleAsync(wg, errChans[i], roleName)
	}
	wg.Wait()

	// Collect all the errors from the async delete calls into a single error struct.
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Debugf("[Failed] %s", err)
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}

	for _, roleName := range roleNames {
		logging.Debugf("[OK] IAM Role %s was deleted", aws.ToString(roleName))
	}
	return nil
}

func (ir *IAMRoles) shouldInclude(iamRole *types.Role, configObj config.Config) bool {
	if iamRole == nil {
		return false
	}

	// The OrganizationAccountAccessRole is a special role that is created by AWS Organizations, and is used to allow
	// users to access the AWS account. We should not delete this role, so we can filter it out of the Roles found and
	// managed by cloud-nuke.
	if strings.Contains(aws.ToString(iamRole.RoleName), "OrganizationAccountAccessRole") {
		return false
	}

	// The ARNs of AWS-reserved IAM roles, which can only be modified or deleted by AWS, contain "aws-reserved", so we can filter them out
	// of the Roles found and managed by cloud-nuke
	if strings.Contains(aws.ToString(iamRole.Arn), "aws-reserved") {
		return false
	}

	return configObj.IAMRoles.ShouldInclude(config.ResourceValue{
		Name: iamRole.RoleName,
		Time: iamRole.CreateDate,
	})
}

func (ir *IAMRoles) deleteIamRoleAsync(wg *sync.WaitGroup, errChan chan error, roleName *string) {
	defer wg.Done()

	var result *multierror.Error

	// Functions used to really nuke an IAM Role as a role can have many attached
	// items we need delete/detach them before actually deleting it.
	// NOTE: The actual role deletion should always be the last one. This way we
	// can guarantee that it will fail if we forgot to delete/detach an item.
	functions := []func(roleName *string) error{
		ir.deleteInstanceProfilesFromRole,
		ir.deleteInlineRolePolicies,
		ir.deleteManagedRolePolicies,
		ir.deleteIamRole,
	}

	for _, fn := range functions {
		if err := fn(roleName); err != nil {
			result = multierror.Append(result, err)
		}
	}

	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.ToString(roleName),
		ResourceType: "IAM Role",
		Error:        result.ErrorOrNil(),
	}
	report.Record(e)

	errChan <- result.ErrorOrNil()
}

// Custom errors

type TooManyIamRoleErr struct{}

func (err TooManyIamRoleErr) Error() string {
	return "Too many IAM Roles requested at once"
}
