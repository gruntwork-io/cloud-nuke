package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

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

// IAMPolicies - represents all IAM Policies on the AWS account
type IAMPolicies struct {
	BaseAwsResource
	Client     IAMPoliciesAPI
	PolicyArns []string
}

func (ip *IAMPolicies) Init(cfg aws.Config) {
	ip.Client = iam.NewFromConfig(cfg)
}

// ResourceName - the simple name of the AWS resource
func (ip *IAMPolicies) ResourceName() string {
	return "iam-policy"
}

// ResourceIdentifiers - The IAM GroupNames
func (ip *IAMPolicies) ResourceIdentifiers() []string {
	return ip.PolicyArns
}

// MaxBatchSize Tentative batch size to ensure AWS doesn't throttle
func (ip *IAMPolicies) MaxBatchSize() int {
	return 20
}

func (ip *IAMPolicies) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.IAMPolicies
}

func (ip *IAMPolicies) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ip.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ip.PolicyArns = aws.ToStringSlice(identifiers)
	return ip.PolicyArns, nil
}

// Nuke - Destroy every group in this collection
func (ip *IAMPolicies) Nuke(ctx context.Context, identifiers []string) error {
	if err := ip.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
