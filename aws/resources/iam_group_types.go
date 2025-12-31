package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

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

// IAMGroups - represents all IAMGroups on the AWS Account
type IAMGroups struct {
	BaseAwsResource
	Client     IAMGroupsAPI
	GroupNames []string
}

func (ig *IAMGroups) Init(cfg aws.Config) {
	ig.Client = iam.NewFromConfig(cfg)
}

// ResourceName - the simple name of the AWS resource
func (ig *IAMGroups) ResourceName() string {
	return "iam-group"
}

// ResourceIdentifiers - The IAM GroupNames
func (ig *IAMGroups) ResourceIdentifiers() []string {
	return ig.GroupNames
}

// Tentative batch size to ensure AWS doesn't throttle
// There's a global max of 500 groups so it shouldn't take long either way
func (ig *IAMGroups) MaxBatchSize() int {
	return 49
}

func (ig *IAMGroups) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.IAMGroups
}

func (ig *IAMGroups) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ig.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ig.GroupNames = aws.ToStringSlice(identifiers)
	return ig.GroupNames, nil
}

// Nuke - Destroy every group in this collection
func (ig *IAMGroups) Nuke(ctx context.Context, identifiers []string) error {
	if err := ig.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
