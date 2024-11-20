package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

func (m mockedIPAMCustomAllocations) DescribeIpamPools(ctx context.Context, params *ec2.DescribeIpamPoolsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamPoolsOutput, error) {
	return &m.DescribeIpamPoolsOutput, nil
}
func (m mockedIPAMCustomAllocations) GetIpamPoolAllocations(ctx context.Context, params *ec2.GetIpamPoolAllocationsInput, optFns ...func(*ec2.Options)) (*ec2.GetIpamPoolAllocationsOutput, error) {
	return &m.GetIpamPoolAllocationsOutput, nil
}

func (m mockedIPAMCustomAllocations) ReleaseIpamPoolAllocation(ctx context.Context, params *ec2.ReleaseIpamPoolAllocationInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseIpamPoolAllocationOutput, error) {
	return &m.ReleaseIpamPoolAllocationOutput, nil
}

type mockedIPAMCustomAllocations struct {
	EC2IPAMCustomAllocationAPI
	DescribeIpamPoolsOutput         ec2.DescribeIpamPoolsOutput
	GetIpamPoolAllocationsOutput    ec2.GetIpamPoolAllocationsOutput
	ReleaseIpamPoolAllocationOutput ec2.ReleaseIpamPoolAllocationOutput
}

func TestIPAMCustomAllocation_GetAll(t *testing.T) {
	t.Parallel()

	var (
		testId1 = "ipam-pool-alloc-0a3b799e43ee044cf8750f69b8329ee27"
		testId2 = "ipam-pool-alloc-0a3b799e43ee044cf8750f69b8329ee28"

		testPool1 = "ipam-pool-0a3b799e43ee044cf8750f69b8329ee28"
	)

	ipam := EC2IPAMCustomAllocation{
		Client: mockedIPAMCustomAllocations{
			DescribeIpamPoolsOutput: ec2.DescribeIpamPoolsOutput{
				IpamPools: []types.IpamPool{
					{
						IpamPoolId: aws.String(testPool1),
					},
				},
			},
			GetIpamPoolAllocationsOutput: ec2.GetIpamPoolAllocationsOutput{
				IpamPoolAllocations: []types.IpamPoolAllocation{
					{

						Cidr:                 aws.String("10.0.0.0/24"),
						IpamPoolAllocationId: aws.String(testId1),
						ResourceType:         types.IpamPoolAllocationResourceTypeCustom,
					},
					{
						Cidr:                 aws.String("10.0.0.16/24"),
						IpamPoolAllocationId: aws.String(testId2),
						ResourceType:         types.IpamPoolAllocationResourceTypeCustom,
					},
				},
			},
		},
		PoolAndAllocationMap: map[string]string{},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testId1, testId2},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := ipam.getAll(context.Background(), config.Config{
				EC2IPAM: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestIPAMCustomAllocation_NukeAll(t *testing.T) {
	t.Parallel()

	var (
		testId1 = "ipam-pool-alloc-0a3b799e43ee044cf8750f69b8329ee27"
		testId2 = "ipam-pool-alloc-0a3b799e43ee044cf8750f69b8329ee28"

		testPool1 = "ipam-pool-0a3b799e43ee044cf8750f69b8329ee28"
	)

	ipam := EC2IPAMCustomAllocation{
		Client: mockedIPAMCustomAllocations{
			DescribeIpamPoolsOutput: ec2.DescribeIpamPoolsOutput{
				IpamPools: []types.IpamPool{
					{
						IpamPoolId: aws.String(testPool1),
					},
				},
			},
			GetIpamPoolAllocationsOutput: ec2.GetIpamPoolAllocationsOutput{
				IpamPoolAllocations: []types.IpamPoolAllocation{
					{

						Cidr:                 aws.String("10.0.0.0/24"),
						IpamPoolAllocationId: aws.String(testId1),
						ResourceType:         types.IpamPoolAllocationResourceTypeCustom,
					},
					{
						Cidr:                 aws.String("10.0.0.16/24"),
						IpamPoolAllocationId: aws.String(testId2),
						ResourceType:         types.IpamPoolAllocationResourceTypeCustom,
					},
				},
			},
		},
		PoolAndAllocationMap: map[string]string{
			"test": testId1,
		},
	}
	err := ipam.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
