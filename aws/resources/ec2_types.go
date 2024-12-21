package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EC2InstancesAPI interface {
	DescribeInstanceAttribute(ctx context.Context, params *ec2.DescribeInstanceAttributeInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceAttributeOutput, error)
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	DescribeAddresses(ctx context.Context, params *ec2.DescribeAddressesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error)
	ReleaseAddress(ctx context.Context, params *ec2.ReleaseAddressInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseAddressOutput, error)
	TerminateInstances(ctx context.Context, params *ec2.TerminateInstancesInput, optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error)
}

// EC2Instances - represents all ec2 instances
type EC2Instances struct {
	BaseAwsResource
	Client      EC2InstancesAPI
	Region      string
	InstanceIds []string
}

func (ei *EC2Instances) InitV2(cfg aws.Config) {
	ei.Client = ec2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (ei *EC2Instances) ResourceName() string {
	return "ec2"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (ei *EC2Instances) ResourceIdentifiers() []string {
	return ei.InstanceIds
}

func (ei *EC2Instances) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (ei *EC2Instances) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EC2
}

func (ei *EC2Instances) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ei.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ei.InstanceIds = aws.ToStringSlice(identifiers)
	return ei.InstanceIds, nil
}

// Nuke - nuke 'em all!!!
func (ei *EC2Instances) Nuke(identifiers []string) error {
	if err := ei.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
