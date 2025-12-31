package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EC2SubnetAPI interface {
	DeleteSubnet(ctx context.Context, params *ec2.DeleteSubnetInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSubnetOutput, error)
	DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
}

// Ec2Subnet- represents all Subnets
type EC2Subnet struct {
	BaseAwsResource
	Client  EC2SubnetAPI
	Region  string
	Subnets []string
}

func (es *EC2Subnet) Init(cfg aws.Config) {
	es.Client = ec2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (es *EC2Subnet) ResourceName() string {
	return "ec2-subnet"
}

func (es *EC2Subnet) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids of the subnets
func (es *EC2Subnet) ResourceIdentifiers() []string {
	return es.Subnets
}

func (es *EC2Subnet) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := es.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	es.Subnets = aws.ToStringSlice(identifiers)
	return es.Subnets, nil
}

// Nuke - nuke 'em all!!!
func (es *EC2Subnet) Nuke(ctx context.Context, identifiers []string) error {
	if err := es.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
