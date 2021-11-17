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

	var roleNames []*string

	// TODO: Probably use ListRoles together with ListRolesPages in case there are lots of roles
	output, err := svc.ListRoles(&iam.ListRolesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	for _, role := range output.Roles {
		if strings.Contains(aws.StringValue(role.RoleName), "OrganizationAccountAccessRole") {
			continue
		}
		if strings.Contains(aws.StringValue(role.Arn), "aws-service-role") || strings.Contains(aws.StringValue(role.Arn), "aws-reserved") {
			continue
		}

		if config.ShouldInclude(aws.StringValue(role.RoleName), configObj.IAMRoles.IncludeRule.NamesRegExp, configObj.IAMRoles.ExcludeRule.NamesRegExp) && excludeAfter.After(*role.CreateDate) {
			roleNames = append(roleNames, role.RoleName)
		}
	}

	return roleNames, nil
}

func detachRolePolicies(svc *iam.IAM, roleName *string) error {
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

func removeRoleFromInstanceProfiles(svc *iam.IAM, roleName *string) error {
	resp, err := svc.ListInstanceProfilesForRole(&iam.ListInstanceProfilesForRoleInput{
		RoleName: roleName,
	})
	if err != nil {
		logging.Logger.Errorf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	for _, profile := range resp.InstanceProfiles {
		_, err := svc.RemoveRoleFromInstanceProfile(&iam.RemoveRoleFromInstanceProfileInput{
			RoleName:            roleName,
			InstanceProfileName: profile.InstanceProfileName,
		})
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}

		logging.Logger.Infof("Removed Role %s from Instance Profile %s", aws.StringValue(roleName), aws.StringValue(profile.InstanceProfileName))
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
		detachRolePolicies,
		deleteInlineRolePolicies,
		removeRoleFromInstanceProfiles,
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
		logging.Logger.Info("No IAM Users to nuke")
		return nil
	}

	logging.Logger.Info("Deleting all IAM Users")

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
