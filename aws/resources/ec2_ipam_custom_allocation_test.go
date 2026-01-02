package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
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

	// Reset global state before test
	customAllocationState.mu.Lock()
	customAllocationState.poolAndAllocationMap = make(map[string]allocationInfo)
	customAllocationState.mu.Unlock()

	ids, err := listEC2IPAMCustomAllocations(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.Equal(t, []string{testId1, testId2}, aws.ToStringSlice(ids))

	// Verify state was stored
	customAllocationState.mu.RLock()
	info1, ok1 := customAllocationState.poolAndAllocationMap[testId1]
	info2, ok2 := customAllocationState.poolAndAllocationMap[testId2]
	customAllocationState.mu.RUnlock()

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

	// Set up state before test
	customAllocationState.mu.Lock()
	customAllocationState.poolAndAllocationMap[testId] = allocationInfo{
		PoolID: testPool,
		Cidr:   testCidr,
	}
	customAllocationState.mu.Unlock()

	mock := &mockEC2IPAMCustomAllocationClient{}
	err := deleteEC2IPAMCustomAllocation(context.Background(), mock, aws.String(testId))
	require.NoError(t, err)
}
