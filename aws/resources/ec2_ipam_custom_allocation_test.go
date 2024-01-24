package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedIPAMCustomAllocations struct {
	ec2iface.EC2API
	GetIpamPoolAllocationsOutput    ec2.GetIpamPoolAllocationsOutput
	ReleaseIpamPoolAllocationOutput ec2.ReleaseIpamPoolAllocationOutput
	DescribeIpamPoolsOutput         ec2.DescribeIpamPoolsOutput
}

func (m mockedIPAMCustomAllocations) GetIpamPoolAllocationsPages(input *ec2.GetIpamPoolAllocationsInput, callback func(*ec2.GetIpamPoolAllocationsOutput, bool) bool) error {
	callback(&m.GetIpamPoolAllocationsOutput, true)
	return nil
}

func (m mockedIPAMCustomAllocations) DescribeIpamPoolsPages(input *ec2.DescribeIpamPoolsInput, callback func(*ec2.DescribeIpamPoolsOutput, bool) bool) error {
	callback(&m.DescribeIpamPoolsOutput, true)
	return nil
}

func (m mockedIPAMCustomAllocations) GetIpamPoolAllocations(params *ec2.GetIpamPoolAllocationsInput) (*ec2.GetIpamPoolAllocationsOutput, error) {
	return &m.GetIpamPoolAllocationsOutput, nil
}

func (m mockedIPAMCustomAllocations) ReleaseIpamPoolAllocation(params *ec2.ReleaseIpamPoolAllocationInput) (*ec2.ReleaseIpamPoolAllocationOutput, error) {
	return &m.ReleaseIpamPoolAllocationOutput, nil
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
				IpamPools: []*ec2.IpamPool{
					{
						IpamPoolId: aws.String(testPool1),
					},
				},
			},
			GetIpamPoolAllocationsOutput: ec2.GetIpamPoolAllocationsOutput{
				IpamPoolAllocations: []*ec2.IpamPoolAllocation{
					{

						Cidr:                 aws.String("10.0.0.0/24"),
						IpamPoolAllocationId: aws.String(testId1),
						ResourceType:         aws.String("custom"),
					},
					{
						Cidr:                 aws.String("10.0.0.16/24"),
						IpamPoolAllocationId: aws.String(testId2),
						ResourceType:         aws.String("custom"),
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
			require.Equal(t, tc.expected, awsgo.StringValueSlice(ids))
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
				IpamPools: []*ec2.IpamPool{
					{
						IpamPoolId: aws.String(testPool1),
					},
				},
			},
			GetIpamPoolAllocationsOutput: ec2.GetIpamPoolAllocationsOutput{
				IpamPoolAllocations: []*ec2.IpamPoolAllocation{
					{

						Cidr:                 aws.String("10.0.0.0/24"),
						IpamPoolAllocationId: aws.String(testId1),
						ResourceType:         aws.String("custom"),
					},
					{
						Cidr:                 aws.String("10.0.0.16/24"),
						IpamPoolAllocationId: aws.String(testId2),
						ResourceType:         aws.String("custom"),
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
