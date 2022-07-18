package aws

import (
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// List all IAM users in the AWS account and returns a slice of the UserNames
func getAllIamRoles(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := iam.New(session)

	allIAMRoles := []*string{}
	err := svc.ListRolesPages(
		&iam.ListRolesInput{},
		func(page *iam.ListRolesOutput, lastPage bool) bool {
			for _, iamRole := range page.Roles {
				if shouldIncludeIAMRole(iamRole, excludeAfter, configObj) {
					allIAMRoles = append(allIAMRoles, iamRole.RoleName)
				}
			}
			return !lastPage
		},
	)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}
	return allIAMRoles, nil
}

func deleteManagedRolePolicies(svc *iam.IAM, roleName *string) error {
	policiesOutput, err := svc.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
		RoleName: roleName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	for _, attachedPolicy := range policiesOutput.AttachedPolicies {
		arn := attachedPolicy.PolicyArn
		_, err = svc.DetachRolePolicy(&iam.DetachRolePolicyInput{
			PolicyArn: arn,
			RoleName:  roleName,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
		logging.Logger.Infof("Detached Policy %s from Role %s", aws.StringValue(arn), aws.StringValue(roleName))
	}

	return nil
}

func deleteInlineRolePolicies(svc *iam.IAM, roleName *string) error {
	policyOutput, err := svc.ListRolePolicies(&iam.ListRolePoliciesInput{
		RoleName: roleName,
	})
	if err != nil {
		logging.Logger.Errorf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	for _, policyName := range policyOutput.PolicyNames {
		_, err := svc.DeleteRolePolicy(&iam.DeleteRolePolicyInput{
			PolicyName: policyName,
			RoleName:   roleName,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
		logging.Logger.Infof("Deleted Inline Policy %s from Role %s", aws.StringValue(policyName), aws.StringValue(roleName))
	}

	return nil
}

func detachInstanceProfilesFromRole(svc *iam.IAM, roleName *string) error {
	profilesOutput, err := svc.ListInstanceProfilesForRole(&iam.ListInstanceProfilesForRoleInput{
		RoleName: roleName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	for _, profile := range profilesOutput.InstanceProfiles {
		_, err := svc.RemoveRoleFromInstanceProfile(&iam.RemoveRoleFromInstanceProfileInput{
			InstanceProfileName: profile.InstanceProfileName,
			RoleName:            roleName,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
		logging.Logger.Infof("Detached InstanceProfile %s from Role %s", aws.StringValue(profile.InstanceProfileName), aws.StringValue(roleName))
	}
	return nil
}

func deleteIamRole(svc *iam.IAM, roleName *string) error {
	_, err := svc.DeleteRole(&iam.DeleteRoleInput{
		RoleName: roleName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// Nuke a single user
func nukeRole(svc *iam.IAM, roleName *string) error {
	// Functions used to really nuke an IAM Role as a role can have many attached
	// items we need delete/detach them before actually deleting it.
	// NOTE: The actual role deletion should always be the last one. This way we
	// can guarantee that it will fail if we forgot to delete/detach an item.
	functions := []func(svc *iam.IAM, roleName *string) error{
		detachInstanceProfilesFromRole,
		deleteInlineRolePolicies,
		deleteManagedRolePolicies,
		deleteIamRole,
	}

	for _, fn := range functions {
		if err := fn(svc, roleName); err != nil {
			return err
		}
	}

	return nil
}

// Delete all IAM Roles
func nukeAllIamRoles(session *session.Session, roleNames []*string) error {
	if len(roleNames) == 0 {
		logging.Logger.Info("No IAM Roles to nuke")
		return nil
	}

	logging.Logger.Info("Deleting all IAM Roles")

	deletedUsers := 0
	svc := iam.New(session)
	multiErr := new(multierror.Error)

	for _, roleName := range roleNames {
		err := nukeRole(svc, roleName)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			multierror.Append(multiErr, err)
		} else {
			deletedUsers++
			logging.Logger.Infof("Deleted IAM Role: %s", *roleName)
		}
	}

	logging.Logger.Infof("[OK] %d IAM Roles(s) terminated", deletedUsers)
	return multiErr.ErrorOrNil()
}

func shouldIncludeIAMRole(iamRole *iam.Role, excludeAfter time.Time, configObj config.Config) bool {
	if iamRole == nil {
		return false
	}

	if strings.Contains(aws.StringValue(iamRole.RoleName), "OrganizationAccountAccessRole") {
		return false
	}

	// The arns of AWS-managed IAM roles, which can only be modified or deleted by AWS, contain "aws-service-role", so we can filter them out
	// of the Roles found and managed by cloud-nuke
	// The same general rule applies with roles whose arn contains "aws-reserved"
	if strings.Contains(aws.StringValue(iamRole.Arn), "aws-service-role") || strings.Contains(aws.StringValue(iamRole.Arn), "aws-reserved") {
		return false
	}

	if excludeAfter.Before(*iamRole.CreateDate) {
		return false
	}

	return config.ShouldInclude(aws.StringValue(iamRole.RoleName), configObj.IAMRoles.IncludeRule.NamesRegExp, configObj.IAMRoles.ExcludeRule.NamesRegExp)
}
