package resources

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// IAMRolesAPI defines the interface for IAM role operations.
type IAMRolesAPI interface {
	ListAttachedRolePolicies(ctx context.Context, params *iam.ListAttachedRolePoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedRolePoliciesOutput, error)
	ListInstanceProfilesForRole(ctx context.Context, params *iam.ListInstanceProfilesForRoleInput, optFns ...func(*iam.Options)) (*iam.ListInstanceProfilesForRoleOutput, error)
	ListRolePolicies(ctx context.Context, params *iam.ListRolePoliciesInput, optFns ...func(*iam.Options)) (*iam.ListRolePoliciesOutput, error)
	ListRoles(ctx context.Context, params *iam.ListRolesInput, optFns ...func(*iam.Options)) (*iam.ListRolesOutput, error)
	ListRoleTags(ctx context.Context, params *iam.ListRoleTagsInput, optFns ...func(*iam.Options)) (*iam.ListRoleTagsOutput, error)
	DeleteInstanceProfile(ctx context.Context, params *iam.DeleteInstanceProfileInput, optFns ...func(*iam.Options)) (*iam.DeleteInstanceProfileOutput, error)
	DetachRolePolicy(ctx context.Context, params *iam.DetachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DetachRolePolicyOutput, error)
	DeleteRolePolicy(ctx context.Context, params *iam.DeleteRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DeleteRolePolicyOutput, error)
	DeleteRole(ctx context.Context, params *iam.DeleteRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteRoleOutput, error)
	RemoveRoleFromInstanceProfile(ctx context.Context, params *iam.RemoveRoleFromInstanceProfileInput, optFns ...func(*iam.Options)) (*iam.RemoveRoleFromInstanceProfileOutput, error)
}

// NewIAMRoles creates a new IAMRoles resource using the generic resource pattern.
func NewIAMRoles() AwsResource {
	return NewAwsResource(&resource.Resource[IAMRolesAPI]{
		ResourceTypeName: "iam-role",
		BatchSize:        20,
		IsGlobal:         true,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[IAMRolesAPI], cfg aws.Config) {
			r.Scope.Region = "global"
			r.Client = iam.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.IAMRoles
		},
		Lister: listIAMRoles,
		Nuker:  resource.SequentialDeleter(deleteIAMRole),
	})
}

// listIAMRoles retrieves all IAM roles that match the config filters.
func listIAMRoles(ctx context.Context, client IAMRolesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var allIAMRoles []*string

	paginator := iam.NewListRolesPaginator(client, &iam.ListRolesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, iamRole := range page.Roles {
			// Always fetch tags to support tag-based filtering, including the default cloud-nuke-excluded tag.
			// This ensures that roles with the exclusion tag are properly filtered out even when no explicit
			// tag filters are configured in the config file.
			tagsOut, err := client.ListRoleTags(ctx, &iam.ListRoleTagsInput{RoleName: iamRole.RoleName})
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			tags := tagsOut.Tags

			if shouldIncludeIAMRole(&iamRole, cfg, tags) {
				allIAMRoles = append(allIAMRoles, iamRole.RoleName)
			}
		}
	}

	return allIAMRoles, nil
}

// shouldIncludeIAMRole determines if an IAM role should be included for deletion.
func shouldIncludeIAMRole(iamRole *types.Role, cfg config.ResourceType, tags []types.Tag) bool {
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

	// The IAM roles with names starting with "AWSServiceRoleFor" are AWS Service-Linked Roles.
	// These are automatically created and managed by AWS services and cannot be manually deleted or modified.
	// Hence, we filter them out from cloud-nuke operations.
	if strings.HasPrefix(aws.ToString(iamRole.RoleName), "AWSServiceRoleFor") {
		logging.Debugf("Filtering out service linked role %s", aws.ToString(iamRole.RoleName))
		return false
	}

	return cfg.ShouldInclude(config.ResourceValue{
		Name: iamRole.RoleName,
		Time: iamRole.CreateDate,
		Tags: util.ConvertIAMTagsToMap(tags),
	})
}

// deleteIAMRole deletes a single IAM role and all its dependencies.
// This function handles:
// 1. Deleting instance profiles from the role
// 2. Deleting inline policies
// 3. Detaching managed policies
// 4. Deleting the role itself
func deleteIAMRole(ctx context.Context, client IAMRolesAPI, roleName *string) error {
	// Delete instance profiles from role
	if err := deleteInstanceProfilesFromRole(ctx, client, roleName); err != nil {
		return err
	}

	// Delete inline policies
	if err := deleteInlineRolePolicies(ctx, client, roleName); err != nil {
		return err
	}

	// Detach managed policies
	if err := detachManagedRolePolicies(ctx, client, roleName); err != nil {
		return err
	}

	// Delete the role
	_, err := client.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: roleName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// deleteInstanceProfilesFromRole removes the role from all instance profiles and deletes them.
func deleteInstanceProfilesFromRole(ctx context.Context, client IAMRolesAPI, roleName *string) error {
	paginator := iam.NewListInstanceProfilesForRolePaginator(client, &iam.ListInstanceProfilesForRoleInput{
		RoleName: roleName,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, profile := range page.InstanceProfiles {
			// Role needs to be removed from instance profile before it can be deleted
			_, err := client.RemoveRoleFromInstanceProfile(ctx, &iam.RemoveRoleFromInstanceProfileInput{
				InstanceProfileName: profile.InstanceProfileName,
				RoleName:            roleName,
			})
			if err != nil {
				return errors.WithStackTrace(err)
			}

			_, err = client.DeleteInstanceProfile(ctx, &iam.DeleteInstanceProfileInput{
				InstanceProfileName: profile.InstanceProfileName,
			})
			if err != nil {
				return errors.WithStackTrace(err)
			}
			logging.Debugf("Detached and Deleted InstanceProfile %s from Role %s", aws.ToString(profile.InstanceProfileName), aws.ToString(roleName))
		}
	}

	return nil
}

// deleteInlineRolePolicies deletes all inline policies attached to the role.
func deleteInlineRolePolicies(ctx context.Context, client IAMRolesAPI, roleName *string) error {
	paginator := iam.NewListRolePoliciesPaginator(client, &iam.ListRolePoliciesInput{
		RoleName: roleName,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, policyName := range page.PolicyNames {
			_, err := client.DeleteRolePolicy(ctx, &iam.DeleteRolePolicyInput{
				PolicyName: aws.String(policyName),
				RoleName:   roleName,
			})
			if err != nil {
				return errors.WithStackTrace(err)
			}
			logging.Debugf("Deleted Inline Policy %s from Role %s", policyName, aws.ToString(roleName))
		}
	}

	return nil
}

// detachManagedRolePolicies detaches all managed policies from the role.
func detachManagedRolePolicies(ctx context.Context, client IAMRolesAPI, roleName *string) error {
	paginator := iam.NewListAttachedRolePoliciesPaginator(client, &iam.ListAttachedRolePoliciesInput{
		RoleName: roleName,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, attachedPolicy := range page.AttachedPolicies {
			_, err = client.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
				PolicyArn: attachedPolicy.PolicyArn,
				RoleName:  roleName,
			})
			if err != nil {
				return errors.WithStackTrace(err)
			}
			logging.Debugf("Detached Policy %s from Role %s", aws.ToString(attachedPolicy.PolicyArn), aws.ToString(roleName))
		}
	}

	return nil
}
