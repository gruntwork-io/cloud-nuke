package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EIPAddressesAPI interface {
	ReleaseAddress(ctx context.Context, params *ec2.ReleaseAddressInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseAddressOutput, error)
	DescribeAddresses(ctx context.Context, params *ec2.DescribeAddressesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error)
}

// EIPAddresses - represents all ebs volumes
type EIPAddresses struct {
	BaseAwsResource
	Client        EIPAddressesAPI
	Region        string
	AllocationIds []string
}

func (eip *EIPAddresses) InitV2(cfg aws.Config) {
	eip.Client = ec2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (eip *EIPAddresses) ResourceName() string {
	return "eip"
}

// ResourceIdentifiers - The instance ids of the eip addresses
func (eip *EIPAddresses) ResourceIdentifiers() []string {
	return eip.AllocationIds
}

func (eip *EIPAddresses) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (eip *EIPAddresses) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.ElasticIP
}

func (eip *EIPAddresses) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := eip.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	eip.AllocationIds = aws.ToStringSlice(identifiers)
	return eip.AllocationIds, nil
}

// Nuke - nuke 'em all!!!
func (eip *EIPAddresses) Nuke(identifiers []string) error {
	if err := eip.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
