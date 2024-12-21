package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EC2IPAMPoolAPI interface {
	DescribeIpamPools(ctx context.Context, params *ec2.DescribeIpamPoolsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamPoolsOutput, error)
	DeleteIpamPool(ctx context.Context, params *ec2.DeleteIpamPoolInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamPoolOutput, error)
}

// EC2IPAMPool IPAM Pool- represents all IPAMs
type EC2IPAMPool struct {
	BaseAwsResource
	Client EC2IPAMPoolAPI
	Region string
	Pools  []string
}

func (pool *EC2IPAMPool) Init(cfg aws.Config) {
	pool.Client = ec2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (pool *EC2IPAMPool) ResourceName() string {
	return "ipam-pool"
}

func (pool *EC2IPAMPool) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids of the IPAMs
func (pool *EC2IPAMPool) ResourceIdentifiers() []string {
	return pool.Pools
}

func (pool *EC2IPAMPool) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EC2IPAMPool
}

func (pool *EC2IPAMPool) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := pool.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	pool.Pools = aws.ToStringSlice(identifiers)
	return pool.Pools, nil
}

// Nuke - nuke 'em all!!!
func (pool *EC2IPAMPool) Nuke(identifiers []string) error {
	if err := pool.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
