package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// IAMPoliciesAPI defines the interface for IAM policy operations.
type IAMPoliciesAPI interface {
	ListEntitiesForPolicy(ctx context.Context, params *iam.ListEntitiesForPolicyInput, optFns ...func(*iam.Options)) (*iam.ListEntitiesForPolicyOutput, error)
	ListPolicies(ctx context.Context, params *iam.ListPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListPoliciesOutput, error)
	ListPolicyTags(ctx context.Context, params *iam.ListPolicyTagsInput, optFns ...func(*iam.Options)) (*iam.ListPolicyTagsOutput, error)
	ListPolicyVersions(ctx context.Context, params *iam.ListPolicyVersionsInput, optFns ...func(*iam.Options)) (*iam.ListPolicyVersionsOutput, error)
	DeletePolicy(ctx context.Context, params *iam.DeletePolicyInput, optFns ...func(*iam.Options)) (*iam.DeletePolicyOutput, error)
	DeletePolicyVersion(ctx context.Context, params *iam.DeletePolicyVersionInput, optFns ...func(*iam.Options)) (*iam.DeletePolicyVersionOutput, error)
	DetachGroupPolicy(ctx context.Context, params *iam.DetachGroupPolicyInput, optFns ...func(*iam.Options)) (*iam.DetachGroupPolicyOutput, error)
	DetachUserPolicy(ctx context.Context, params *iam.DetachUserPolicyInput, optFns ...func(*iam.Options)) (*iam.DetachUserPolicyOutput, error)
	DetachRolePolicy(ctx context.Context, params *iam.DetachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DetachRolePolicyOutput, error)
	DeleteUserPermissionsBoundary(ctx context.Context, params *iam.DeleteUserPermissionsBoundaryInput, optFns ...func(*iam.Options)) (*iam.DeleteUserPermissionsBoundaryOutput, error)
	DeleteRolePermissionsBoundary(ctx context.Context, params *iam.DeleteRolePermissionsBoundaryInput, optFns ...func(*iam.Options)) (*iam.DeleteRolePermissionsBoundaryOutput, error)
}

// NewIAMPolicies creates a new IAM Policies resource using the generic resource pattern.
func NewIAMPolicies() AwsResource {
	return NewAwsResource(&resource.Resource[IAMPoliciesAPI]{
		ResourceTypeName: "iam-policy",
		BatchSize:        20,
		IsGlobal:         true,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[IAMPoliciesAPI], cfg aws.Config) {
			r.Scope.Region = "global"
			r.Client = iam.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.IAMPolicies
		},
		Lister: listIAMPolicies,
		// IAM policy deletion requires multiple cleanup steps per policy
		Nuker: resource.SequentialDeleter(deleteIAMPolicy),
	})
}

// listIAMPolicies retrieves all customer-managed IAM policies that match the config filters.
func listIAMPolicies(ctx context.Context, client IAMPoliciesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var allIamPolicies []*string

	paginator := iam.NewListPoliciesPaginator(client, &iam.ListPoliciesInput{Scope: types.PolicyScopeTypeLocal})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, policy := range page.Policies {
			// Always fetch tags to support tag-based filtering, including the default cloud-nuke-excluded tag.
			// This ensures that policies with the exclusion tag are properly filtered out even when no explicit
			// tag filters are configured in the config file.
			tagsOut, err := client.ListPolicyTags(ctx, &iam.ListPolicyTagsInput{PolicyArn: policy.Arn})
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			tags := tagsOut.Tags

			if cfg.ShouldInclude(config.ResourceValue{
				Name: policy.PolicyName,
				Time: policy.CreateDate,
				Tags: util.ConvertIAMTagsToMap(tags),
			}) {
				allIamPolicies = append(allIamPolicies, policy.Arn)
			}
		}
	}

	return allIamPolicies, nil
}

// deleteIAMPolicy deletes a single IAM policy after performing all necessary cleanup:
// 1. Detach permissions boundaries from users/roles
// 2. Detach policy from entities (users/roles/groups)
// 3. Delete non-default policy versions
// 4. Delete the policy itself
func deleteIAMPolicy(ctx context.Context, client IAMPoliciesAPI, policyArn *string) error {
	// Remove the policy as a permissions boundary from any users or roles.
	// This must be done before detaching regular policy attachments, as AWS
	// prevents deletion of policies that are still used as permissions boundaries.
	if err := detachPermissionsBoundaryEntities(ctx, client, policyArn); err != nil {
		return err
	}

	// Detach any entities the policy is attached to as a permissions policy
	if err := detachPolicyEntities(ctx, client, policyArn); err != nil {
		return err
	}

	// Delete old policy versions (non-default versions)
	if err := deleteOldPolicyVersions(ctx, client, policyArn); err != nil {
		return err
	}

	// Delete the policy itself
	_, err := client.DeletePolicy(ctx, &iam.DeletePolicyInput{PolicyArn: policyArn})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	logging.Debugf("[OK] IAM Policy %s was deleted in global", aws.ToString(policyArn))
	return nil
}

// detachPermissionsBoundaryEntities removes the policy as a permissions boundary from any users or roles.
func detachPermissionsBoundaryEntities(ctx context.Context, client IAMPoliciesAPI, policyArn *string) error {
	var allBoundaryRoles []*string
	var allBoundaryUsers []*string

	// List entities where this policy is used as a permissions boundary
	paginator := iam.NewListEntitiesForPolicyPaginator(client, &iam.ListEntitiesForPolicyInput{
		PolicyArn:         policyArn,
		PolicyUsageFilter: types.PolicyUsageTypePermissionsBoundary,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, role := range page.PolicyRoles {
			allBoundaryRoles = append(allBoundaryRoles, role.RoleName)
		}
		for _, user := range page.PolicyUsers {
			allBoundaryUsers = append(allBoundaryUsers, user.UserName)
		}
	}

	// Remove permissions boundary from users
	for _, userName := range allBoundaryUsers {
		_, err := client.DeleteUserPermissionsBoundary(ctx, &iam.DeleteUserPermissionsBoundaryInput{
			UserName: userName,
		})
		if err != nil {
			return errors.WithStackTrace(err)
		}
		logging.Debugf("Removed permissions boundary from user %s", aws.ToString(userName))
	}

	// Remove permissions boundary from roles
	for _, roleName := range allBoundaryRoles {
		_, err := client.DeleteRolePermissionsBoundary(ctx, &iam.DeleteRolePermissionsBoundaryInput{
			RoleName: roleName,
		})
		if err != nil {
			return errors.WithStackTrace(err)
		}
		logging.Debugf("Removed permissions boundary from role %s", aws.ToString(roleName))
	}

	return nil
}

// detachPolicyEntities detaches the policy from all users, roles, and groups.
func detachPolicyEntities(ctx context.Context, client IAMPoliciesAPI, policyArn *string) error {
	var allPolicyGroups []*string
	var allPolicyRoles []*string
	var allPolicyUsers []*string

	paginator := iam.NewListEntitiesForPolicyPaginator(client, &iam.ListEntitiesForPolicyInput{PolicyArn: policyArn})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, group := range page.PolicyGroups {
			allPolicyGroups = append(allPolicyGroups, group.GroupName)
		}
		for _, role := range page.PolicyRoles {
			allPolicyRoles = append(allPolicyRoles, role.RoleName)
		}
		for _, user := range page.PolicyUsers {
			allPolicyUsers = append(allPolicyUsers, user.UserName)
		}
	}

	// Detach policy from any users
	for _, userName := range allPolicyUsers {
		_, err := client.DetachUserPolicy(ctx, &iam.DetachUserPolicyInput{
			UserName:  userName,
			PolicyArn: policyArn,
		})
		if err != nil {
			return errors.WithStackTrace(err)
		}
	}

	// Detach policy from any groups
	for _, groupName := range allPolicyGroups {
		_, err := client.DetachGroupPolicy(ctx, &iam.DetachGroupPolicyInput{
			GroupName: groupName,
			PolicyArn: policyArn,
		})
		if err != nil {
			return errors.WithStackTrace(err)
		}
	}

	// Detach policy from any roles
	for _, roleName := range allPolicyRoles {
		_, err := client.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
			RoleName:  roleName,
			PolicyArn: policyArn,
		})
		if err != nil {
			return errors.WithStackTrace(err)
		}
	}

	return nil
}

// deleteOldPolicyVersions deletes all non-default versions of the policy.
func deleteOldPolicyVersions(ctx context.Context, client IAMPoliciesAPI, policyArn *string) error {
	var versionsToRemove []*string

	paginator := iam.NewListPolicyVersionsPaginator(client, &iam.ListPolicyVersionsInput{PolicyArn: policyArn})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, policyVersion := range page.Versions {
			if !policyVersion.IsDefaultVersion {
				versionsToRemove = append(versionsToRemove, policyVersion.VersionId)
			}
		}
	}

	// Delete old policy versions
	for _, versionId := range versionsToRemove {
		_, err := client.DeletePolicyVersion(ctx, &iam.DeletePolicyVersionInput{
			VersionId: versionId,
			PolicyArn: policyArn,
		})
		if err != nil {
			return errors.WithStackTrace(err)
		}
	}

	return nil
}
