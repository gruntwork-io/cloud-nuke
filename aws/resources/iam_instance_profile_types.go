package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type IAMInstanceProfileAPI interface {
	ListInstanceProfiles(ctx context.Context, params *iam.ListInstanceProfilesInput, optFns ...func(*iam.Options)) (*iam.ListInstanceProfilesOutput, error)
	GetInstanceProfile(ctx context.Context, params *iam.GetInstanceProfileInput, optFns ...func(*iam.Options)) (*iam.GetInstanceProfileOutput, error)
	RemoveRoleFromInstanceProfile(ctx context.Context, params *iam.RemoveRoleFromInstanceProfileInput, optFns ...func(*iam.Options)) (*iam.RemoveRoleFromInstanceProfileOutput, error)
	DeleteInstanceProfile(ctx context.Context, params *iam.DeleteInstanceProfileInput, optFns ...func(*iam.Options)) (*iam.DeleteInstanceProfileOutput, error)
}

// IAMInstanceProfiles - represents all IAM Policies on the AWS account
type IAMInstanceProfiles struct {
	BaseAwsResource
	Client           IAMInstanceProfileAPI
	InstanceProfiles []string
}

func (ip *IAMInstanceProfiles) Init(cfg aws.Config) {
	ip.Client = iam.NewFromConfig(cfg)
}

// ResourceName - the simple name of the AWS resource
func (ip *IAMInstanceProfiles) ResourceName() string {
	return "iam-instance-profile"
}

// ResourceIdentifiers - The IAM Instance Profile Names
func (ip *IAMInstanceProfiles) ResourceIdentifiers() []string {
	return ip.InstanceProfiles
}

// MaxBatchSize Tentative batch size to ensure AWS doesn't throttle
func (ip *IAMInstanceProfiles) MaxBatchSize() int {
	return 20
}

func (ip *IAMInstanceProfiles) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.IAMPolicies
}

func (ip *IAMInstanceProfiles) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ip.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ip.InstanceProfiles = aws.ToStringSlice(identifiers)
	return ip.InstanceProfiles, nil
}

// Nuke - Destroy every instance profiles in this collection
func (ip *IAMInstanceProfiles) Nuke(ctx context.Context, identifiers []string) error {
	if err := ip.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
