package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type IAMRolesAPI interface {
	ListAttachedRolePolicies(ctx context.Context, params *iam.ListAttachedRolePoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedRolePoliciesOutput, error)
	ListInstanceProfilesForRole(ctx context.Context, params *iam.ListInstanceProfilesForRoleInput, optFns ...func(*iam.Options)) (*iam.ListInstanceProfilesForRoleOutput, error)
	ListRolePolicies(ctx context.Context, params *iam.ListRolePoliciesInput, optFns ...func(*iam.Options)) (*iam.ListRolePoliciesOutput, error)
	ListRoles(ctx context.Context, params *iam.ListRolesInput, optFns ...func(*iam.Options)) (*iam.ListRolesOutput, error)
	DeleteInstanceProfile(ctx context.Context, params *iam.DeleteInstanceProfileInput, optFns ...func(*iam.Options)) (*iam.DeleteInstanceProfileOutput, error)
	DetachRolePolicy(ctx context.Context, params *iam.DetachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DetachRolePolicyOutput, error)
	DeleteRolePolicy(ctx context.Context, params *iam.DeleteRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DeleteRolePolicyOutput, error)
	DeleteRole(ctx context.Context, params *iam.DeleteRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteRoleOutput, error)
	RemoveRoleFromInstanceProfile(ctx context.Context, params *iam.RemoveRoleFromInstanceProfileInput, optFns ...func(*iam.Options)) (*iam.RemoveRoleFromInstanceProfileOutput, error)
}

// IAMRoles - represents all IAMRoles on the AWS Account
type IAMRoles struct {
	BaseAwsResource
	Client    IAMRolesAPI
	RoleNames []string
}

func (ir *IAMRoles) InitV2(cfg aws.Config) {
	ir.Client = iam.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (ir *IAMRoles) ResourceName() string {
	return "iam-role"
}

// ResourceIdentifiers - The IAM UserNames
func (ir *IAMRoles) ResourceIdentifiers() []string {
	return ir.RoleNames
}

// Tentative batch size to ensure AWS doesn't throttle
func (ir *IAMRoles) MaxBatchSize() int {
	return 20
}

func (ir *IAMRoles) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.IAMRoles
}

func (ir *IAMRoles) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ir.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ir.RoleNames = aws.ToStringSlice(identifiers)
	return ir.RoleNames, nil
}

// Nuke - nuke 'em all!!!
func (ir *IAMRoles) Nuke(identifiers []string) error {
	if err := ir.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
