package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockDBGlobalClustersClient struct {
	DescribeGlobalClustersOutput rds.DescribeGlobalClustersOutput
	DescribeGlobalClustersError  error
	DeleteGlobalClusterOutput    rds.DeleteGlobalClusterOutput
}

func (m *mockDBGlobalClustersClient) DescribeGlobalClusters(ctx context.Context, params *rds.DescribeGlobalClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeGlobalClustersOutput, error) {
	return &m.DescribeGlobalClustersOutput, m.DescribeGlobalClustersError
}

func (m *mockDBGlobalClustersClient) DeleteGlobalCluster(ctx context.Context, params *rds.DeleteGlobalClusterInput, optFns ...func(*rds.Options)) (*rds.DeleteGlobalClusterOutput, error) {
	return &m.DeleteGlobalClusterOutput, nil
}

func TestListDBGlobalClusters(t *testing.T) {
	t.Parallel()

	testName1 := "test-global-cluster1"
	testName2 := "test-global-cluster2"

	mock := &mockDBGlobalClustersClient{
		DescribeGlobalClustersOutput: rds.DescribeGlobalClustersOutput{
			GlobalClusters: []types.GlobalCluster{
				{GlobalClusterIdentifier: aws.String(testName1)},
				{GlobalClusterIdentifier: aws.String(testName2)},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testName2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listDBGlobalClusters(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteDBGlobalCluster(t *testing.T) {
	t.Parallel()

	mock := &mockDBGlobalClustersClient{
		DeleteGlobalClusterOutput: rds.DeleteGlobalClusterOutput{},
	}

	err := deleteDBGlobalCluster(context.Background(), mock, aws.String("test"))
	require.NoError(t, err)
}

func TestWaitForDBGlobalClustersDeleted(t *testing.T) {
	t.Parallel()

	// Mock returns NotFound error immediately - cluster already deleted
	mock := &mockDBGlobalClustersClient{
		DescribeGlobalClustersError: &types.GlobalClusterNotFoundFault{},
	}

	err := waitForDBGlobalClustersDeleted(context.Background(), mock, []string{"test"})
	require.NoError(t, err)
}
