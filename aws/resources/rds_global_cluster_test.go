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
	TagsByARN                    map[string][]types.Tag
}

func (m *mockDBGlobalClustersClient) DescribeGlobalClusters(ctx context.Context, params *rds.DescribeGlobalClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeGlobalClustersOutput, error) {
	return &m.DescribeGlobalClustersOutput, m.DescribeGlobalClustersError
}

func (m *mockDBGlobalClustersClient) DeleteGlobalCluster(ctx context.Context, params *rds.DeleteGlobalClusterInput, optFns ...func(*rds.Options)) (*rds.DeleteGlobalClusterOutput, error) {
	return &m.DeleteGlobalClusterOutput, nil
}

func (m *mockDBGlobalClustersClient) ListTagsForResource(ctx context.Context, params *rds.ListTagsForResourceInput, optFns ...func(*rds.Options)) (*rds.ListTagsForResourceOutput, error) {
	return &rds.ListTagsForResourceOutput{TagList: m.TagsByARN[aws.ToString(params.ResourceName)]}, nil
}

func TestListDBGlobalClusters(t *testing.T) {
	t.Parallel()

	testName1 := "test-global-cluster1"
	testName2 := "test-global-cluster2"
	testArn1 := "arn:aws:rds::123456789:global-cluster:" + testName1
	testArn2 := "arn:aws:rds::123456789:global-cluster:" + testName2

	mock := &mockDBGlobalClustersClient{
		DescribeGlobalClustersOutput: rds.DescribeGlobalClustersOutput{
			GlobalClusters: []types.GlobalCluster{
				{GlobalClusterIdentifier: aws.String(testName1), GlobalClusterArn: aws.String(testArn1)},
				{GlobalClusterIdentifier: aws.String(testName2), GlobalClusterArn: aws.String(testArn2)},
			},
		},
		TagsByARN: map[string][]types.Tag{
			testArn1: {{Key: aws.String("env"), Value: aws.String("prod")}},
			testArn2: {{Key: aws.String("env"), Value: aws.String("dev")}},
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
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"env": {RE: *regexp.MustCompile("^prod$")},
					},
				},
			},
			expected: []string{testName1},
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
