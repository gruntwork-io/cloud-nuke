package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EC2IPAMCustomAllocationAPI interface {
	DescribeIpamPools(ctx context.Context, params *ec2.DescribeIpamPoolsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamPoolsOutput, error)
	GetIpamPoolAllocations(ctx context.Context, params *ec2.GetIpamPoolAllocationsInput, optFns ...func(*ec2.Options)) (*ec2.GetIpamPoolAllocationsOutput, error)
	ReleaseIpamPoolAllocation(ctx context.Context, params *ec2.ReleaseIpamPoolAllocationInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseIpamPoolAllocationOutput, error)
}

// EC2IPAMCustomAllocation IPAM Byoasn- represents all IPAMs
type EC2IPAMCustomAllocation struct {
	BaseAwsResource
	Client               EC2IPAMCustomAllocationAPI
	Region               string
	Allocations          []string
	PoolAndAllocationMap map[string]string
}

func (cs *EC2IPAMCustomAllocation) InitV2(cfg aws.Config) {
	cs.Client = ec2.NewFromConfig(cfg)
	cs.PoolAndAllocationMap = make(map[string]string)
}

func (cs *EC2IPAMCustomAllocation) IsUsingV2() bool { return true }

// ResourceName - the simple name of the aws resource
func (cs *EC2IPAMCustomAllocation) ResourceName() string {
	return "ipam-custom-allocation"
}

func (cs *EC2IPAMCustomAllocation) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 1000
}

// ResourceIdentifiers - The ids of the IPAMs
func (cs *EC2IPAMCustomAllocation) ResourceIdentifiers() []string {
	return cs.Allocations
}

func (cs *EC2IPAMCustomAllocation) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := cs.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	cs.Allocations = aws.ToStringSlice(identifiers)
	return cs.Allocations, nil
}

// Nuke - nuke 'em all!!!
func (cs *EC2IPAMCustomAllocation) Nuke(identifiers []string) error {
	if err := cs.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
