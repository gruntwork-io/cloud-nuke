package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// IAMGroupsAPI defines the interface for IAM group operations.
type IAMGroupsAPI interface {
	DetachGroupPolicy(ctx context.Context, params *iam.DetachGroupPolicyInput, optFns ...func(*iam.Options)) (*iam.DetachGroupPolicyOutput, error)
	DeleteGroupPolicy(ctx context.Context, params *iam.DeleteGroupPolicyInput, optFns ...func(*iam.Options)) (*iam.DeleteGroupPolicyOutput, error)
	DeleteGroup(ctx context.Context, params *iam.DeleteGroupInput, optFns ...func(*iam.Options)) (*iam.DeleteGroupOutput, error)
	GetGroup(ctx context.Context, params *iam.GetGroupInput, optFns ...func(*iam.Options)) (*iam.GetGroupOutput, error)
	ListAttachedGroupPolicies(ctx context.Context, params *iam.ListAttachedGroupPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedGroupPoliciesOutput, error)
	ListGroups(ctx context.Context, params *iam.ListGroupsInput, optFns ...func(*iam.Options)) (*iam.ListGroupsOutput, error)
	ListGroupPolicies(ctx context.Context, params *iam.ListGroupPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListGroupPoliciesOutput, error)
	RemoveUserFromGroup(ctx context.Context, params *iam.RemoveUserFromGroupInput, optFns ...func(*iam.Options)) (*iam.RemoveUserFromGroupOutput, error)
}

// NewIAMGroups creates a new IAMGroups resource using the generic resource pattern.
func NewIAMGroups() AwsResource {
	return NewAwsResource(&resource.Resource[IAMGroupsAPI]{
		ResourceTypeName: "iam-group",
		BatchSize:        49,
		IsGlobal:         true,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[IAMGroupsAPI], cfg aws.Config) {
			r.Scope.Region = "global"
			r.Client = iam.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.IAMGroups
		},
		Lister: listIAMGroups,
		Nuker:  resource.SequentialDeleter(deleteIAMGroup),
	})
}

// listIAMGroups retrieves all IAM groups that match the config filters.
func listIAMGroups(ctx context.Context, client IAMGroupsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var allIamGroups []*string

	paginator := iam.NewListGroupsPaginator(client, &iam.ListGroupsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, iamGroup := range page.Groups {
			if cfg.ShouldInclude(config.ResourceValue{
				Time: iamGroup.CreateDate,
				Name: iamGroup.GroupName,
			}) {
				allIamGroups = append(allIamGroups, iamGroup.GroupName)
			}
		}
	}

	return allIamGroups, nil
}

// deleteIAMGroup deletes a single IAM group.
// This requires: removing users from group, detaching policies, deleting inline policies, then deleting the group.
func deleteIAMGroup(ctx context.Context, client IAMGroupsAPI, groupName *string) error {
	// Remove any users from the group
	if err := removeUsersFromGroup(ctx, client, groupName); err != nil {
		return err
	}

	// Detach any attached policies on the group
	if err := detachGroupPolicies(ctx, client, groupName); err != nil {
		return err
	}

	// Delete any inline policies on the group
	if err := deleteGroupInlinePolicies(ctx, client, groupName); err != nil {
		return err
	}

	// Delete the group
	_, err := client.DeleteGroup(ctx, &iam.DeleteGroupInput{
		GroupName: groupName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	logging.Debugf("[OK] IAM Group %s was deleted in global", aws.ToString(groupName))
	return nil
}

// removeUsersFromGroup removes all users from the specified IAM group.
func removeUsersFromGroup(ctx context.Context, client IAMGroupsAPI, groupName *string) error {
	grp, err := client.GetGroup(ctx, &iam.GetGroupInput{
		GroupName: groupName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	for _, user := range grp.Users {
		_, err := client.RemoveUserFromGroup(ctx, &iam.RemoveUserFromGroupInput{
			UserName:  user.UserName,
			GroupName: groupName,
		})
		if err != nil {
			return errors.WithStackTrace(err)
		}
	}

	return nil
}

// detachGroupPolicies detaches all attached policies from the specified IAM group.
func detachGroupPolicies(ctx context.Context, client IAMGroupsAPI, groupName *string) error {
	paginator := iam.NewListAttachedGroupPoliciesPaginator(client, &iam.ListAttachedGroupPoliciesInput{GroupName: groupName})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, policy := range page.AttachedPolicies {
			_, err := client.DetachGroupPolicy(ctx, &iam.DetachGroupPolicyInput{
				GroupName: groupName,
				PolicyArn: policy.PolicyArn,
			})
			if err != nil {
				return errors.WithStackTrace(err)
			}
		}
	}

	return nil
}

// deleteGroupInlinePolicies deletes all inline policies from the specified IAM group.
func deleteGroupInlinePolicies(ctx context.Context, client IAMGroupsAPI, groupName *string) error {
	paginator := iam.NewListGroupPoliciesPaginator(client, &iam.ListGroupPoliciesInput{GroupName: groupName})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		for _, policyName := range page.PolicyNames {
			_, err := client.DeleteGroupPolicy(ctx, &iam.DeleteGroupPolicyInput{
				GroupName:  groupName,
				PolicyName: aws.String(policyName),
			})
			if err != nil {
				return errors.WithStackTrace(err)
			}
		}
	}

	return nil
}
