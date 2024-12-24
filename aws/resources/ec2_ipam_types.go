package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EC2IPAMAPIaa interface {
	DescribeIpams(ctx context.Context, params *ec2.DescribeIpamsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamsOutput, error)
	DeleteIpam(ctx context.Context, params *ec2.DeleteIpamInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamOutput, error)
	GetIpamPoolCidrs(ctx context.Context, params *ec2.GetIpamPoolCidrsInput, optFns ...func(*ec2.Options)) (*ec2.GetIpamPoolCidrsOutput, error)
	DeprovisionIpamPoolCidr(ctx context.Context, params *ec2.DeprovisionIpamPoolCidrInput, optFns ...func(*ec2.Options)) (*ec2.DeprovisionIpamPoolCidrOutput, error)
	GetIpamPoolAllocations(ctx context.Context, params *ec2.GetIpamPoolAllocationsInput, optFns ...func(*ec2.Options)) (*ec2.GetIpamPoolAllocationsOutput, error)
	ReleaseIpamPoolAllocation(ctx context.Context, params *ec2.ReleaseIpamPoolAllocationInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseIpamPoolAllocationOutput, error)
	DescribeIpamScopes(ctx context.Context, params *ec2.DescribeIpamScopesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamScopesOutput, error)
	DescribeIpamPools(ctx context.Context, params *ec2.DescribeIpamPoolsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamPoolsOutput, error)
	DeleteIpamPool(ctx context.Context, params *ec2.DeleteIpamPoolInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamPoolOutput, error)
}

// IPAM - represents all IPAMs
type EC2IPAMs struct {
	BaseAwsResource
	Client EC2IPAMAPIaa
	Region string
	IDs    []string
}

func (ipam *EC2IPAMs) Init(cfg aws.Config) {
	ipam.Client = ec2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (ipam *EC2IPAMs) ResourceName() string {
	return "ipam"
}

func (ipam *EC2IPAMs) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids of the IPAMs
func (ipam *EC2IPAMs) ResourceIdentifiers() []string {
	return ipam.IDs
}

func (ipam *EC2IPAMs) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EC2IPAM
}

func (ipam *EC2IPAMs) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := ipam.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	ipam.IDs = aws.ToStringSlice(identifiers)
	return ipam.IDs, nil
}

// Nuke - nuke 'em all!!!
func (ipam *EC2IPAMs) Nuke(identifiers []string) error {
	if err := ipam.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
