package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/require"
)

type mockEC2IPAMCustomAllocationClient struct {
	DescribeIpamPoolsOutput         ec2.DescribeIpamPoolsOutput
	GetIpamPoolAllocationsOutput    ec2.GetIpamPoolAllocationsOutput
	ReleaseIpamPoolAllocationOutput ec2.ReleaseIpamPoolAllocationOutput
}

func (m *mockEC2IPAMCustomAllocationClient) DescribeIpamPools(ctx context.Context, params *ec2.DescribeIpamPoolsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamPoolsOutput, error) {
	return &m.DescribeIpamPoolsOutput, nil
}

func (m *mockEC2IPAMCustomAllocationClient) GetIpamPoolAllocations(ctx context.Context, params *ec2.GetIpamPoolAllocationsInput, optFns ...func(*ec2.Options)) (*ec2.GetIpamPoolAllocationsOutput, error) {
	return &m.GetIpamPoolAllocationsOutput, nil
}

func (m *mockEC2IPAMCustomAllocationClient) ReleaseIpamPoolAllocation(ctx context.Context, params *ec2.ReleaseIpamPoolAllocationInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseIpamPoolAllocationOutput, error) {
	return &m.ReleaseIpamPoolAllocationOutput, nil
}

func TestListEC2IPAMCustomAllocations(t *testing.T) {
	t.Parallel()

	testId1 := "ipam-pool-alloc-0a3b799e43ee044cf8750f69b8329ee27"
	testId2 := "ipam-pool-alloc-0a3b799e43ee044cf8750f69b8329ee28"
	testPool1 := "ipam-pool-0a3b799e43ee044cf8750f69b8329ee28"

	mock := &mockEC2IPAMCustomAllocationClient{
		DescribeIpamPoolsOutput: ec2.DescribeIpamPoolsOutput{
			IpamPools: []types.IpamPool{
				{IpamPoolId: aws.String(testPool1)},
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
	}

	poolAndAllocationMap := make(map[string]allocationInfo)
	ids, err := listEC2IPAMCustomAllocations(context.Background(), mock, poolAndAllocationMap)
	require.NoError(t, err)
	require.Equal(t, []string{testId1, testId2}, aws.ToStringSlice(ids))

	info1, ok1 := poolAndAllocationMap[testId1]
	info2, ok2 := poolAndAllocationMap[testId2]
	require.True(t, ok1)
	require.True(t, ok2)
	require.Equal(t, testPool1, info1.PoolID)
	require.Equal(t, "10.0.0.0/24", info1.Cidr)
	require.Equal(t, testPool1, info2.PoolID)
	require.Equal(t, "10.0.0.16/24", info2.Cidr)
}

func TestDeleteEC2IPAMCustomAllocation(t *testing.T) {
	t.Parallel()

	testId := "ipam-pool-alloc-test"
	testPool := "ipam-pool-test"
	testCidr := "10.0.0.0/24"

	poolAndAllocationMap := map[string]allocationInfo{
		testId: {PoolID: testPool, Cidr: testCidr},
	}

	mock := &mockEC2IPAMCustomAllocationClient{}
	err := deleteEC2IPAMCustomAllocation(context.Background(), mock, aws.String(testId), poolAndAllocationMap)
	require.NoError(t, err)
}

// TestEC2IPAMCustomAllocationInstancesAreIsolated exercises the multi-region scenario that regressed
// in v0.49.0: an allocation discovered in one instance must remain resolvable after a second instance
// is constructed and initialized. A shared package-level map (the pre-fix design) fails this test;
// per-instance closure state passes it.
func TestEC2IPAMCustomAllocationInstancesAreIsolated(t *testing.T) {
	t.Parallel()

	regionAId := "ipam-pool-alloc-region-a"
	regionAPool := "ipam-pool-region-a"
	regionBId := "ipam-pool-alloc-region-b"
	regionBPool := "ipam-pool-region-b"

	mapA := make(map[string]allocationInfo)
	mapB := make(map[string]allocationInfo)

	_, err := listEC2IPAMCustomAllocations(context.Background(), &mockEC2IPAMCustomAllocationClient{
		DescribeIpamPoolsOutput: ec2.DescribeIpamPoolsOutput{
			IpamPools: []types.IpamPool{{IpamPoolId: aws.String(regionAPool)}},
		},
		GetIpamPoolAllocationsOutput: ec2.GetIpamPoolAllocationsOutput{
			IpamPoolAllocations: []types.IpamPoolAllocation{{
				Cidr:                 aws.String("10.1.0.0/24"),
				IpamPoolAllocationId: aws.String(regionAId),
				ResourceType:         types.IpamPoolAllocationResourceTypeCustom,
			}},
		},
	}, mapA)
	require.NoError(t, err)

	_, err = listEC2IPAMCustomAllocations(context.Background(), &mockEC2IPAMCustomAllocationClient{
		DescribeIpamPoolsOutput: ec2.DescribeIpamPoolsOutput{
			IpamPools: []types.IpamPool{{IpamPoolId: aws.String(regionBPool)}},
		},
		GetIpamPoolAllocationsOutput: ec2.GetIpamPoolAllocationsOutput{
			IpamPoolAllocations: []types.IpamPoolAllocation{{
				Cidr:                 aws.String("10.2.0.0/24"),
				IpamPoolAllocationId: aws.String(regionBId),
				ResourceType:         types.IpamPoolAllocationResourceTypeCustom,
			}},
		},
	}, mapB)
	require.NoError(t, err)

	infoA, okA := mapA[regionAId]
	require.True(t, okA, "region A's allocation metadata should survive region B's listing")
	require.Equal(t, regionAPool, infoA.PoolID)

	infoB, okB := mapB[regionBId]
	require.True(t, okB)
	require.Equal(t, regionBPool, infoB.PoolID)

	_, crossA := mapA[regionBId]
	_, crossB := mapB[regionAId]
	require.False(t, crossA, "region A's map should not see region B's allocation")
	require.False(t, crossB, "region B's map should not see region A's allocation")
}
