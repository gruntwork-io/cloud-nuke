package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EC2PlacementGroupsAPI interface {
	DescribePlacementGroups(ctx context.Context, params *ec2.DescribePlacementGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribePlacementGroupsOutput, error)
	DeletePlacementGroup(ctx context.Context, params *ec2.DeletePlacementGroupInput, optFns ...func(*ec2.Options)) (*ec2.DeletePlacementGroupOutput, error)
}

type EC2PlacementGroups struct {
	BaseAwsResource
	Client              EC2PlacementGroupsAPI
	Region              string
	PlacementGroupNames []string
}

func (k *EC2PlacementGroups) InitV2(cfg aws.Config) {
	k.Client = ec2.NewFromConfig(cfg)
}

func (k *EC2PlacementGroups) IsUsingV2() bool { return true }

// ResourceName - the simple name of the aws resource
func (k *EC2PlacementGroups) ResourceName() string {
	return "ec2-placement-groups"
}

// ResourceIdentifiers - IDs of the ec2 key pairs
func (k *EC2PlacementGroups) ResourceIdentifiers() []string {
	return k.PlacementGroupNames
}

func (k *EC2PlacementGroups) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 200
}

func (k *EC2PlacementGroups) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EC2PlacementGroups
}

func (k *EC2PlacementGroups) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := k.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	k.PlacementGroupNames = aws.ToStringSlice(identifiers)
	return k.PlacementGroupNames, nil
}

func (k *EC2PlacementGroups) Nuke(identifiers []string) error {
	if err := k.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
