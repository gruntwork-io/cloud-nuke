package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/aws/aws-sdk-go-v2/service/kafka/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockMSKClient struct {
	ListClustersV2Output kafka.ListClustersV2Output
	DeleteClusterOutput  kafka.DeleteClusterOutput
}

func (m *mockMSKClient) ListClustersV2(ctx context.Context, params *kafka.ListClustersV2Input, optFns ...func(*kafka.Options)) (*kafka.ListClustersV2Output, error) {
	return &m.ListClustersV2Output, nil
}

func (m *mockMSKClient) DeleteCluster(ctx context.Context, params *kafka.DeleteClusterInput, optFns ...func(*kafka.Options)) (*kafka.DeleteClusterOutput, error) {
	return &m.DeleteClusterOutput, nil
}

func TestMSKCluster_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testArn1 := "arn:aws:kafka:us-east-1:123456789012:cluster/test-cluster-1/abc123"
	testArn2 := "arn:aws:kafka:us-east-1:123456789012:cluster/test-cluster-2/def456"
	testName1 := "test-cluster-1"
	testName2 := "test-cluster-2"

	mock := &mockMSKClient{
		ListClustersV2Output: kafka.ListClustersV2Output{
			ClusterInfoList: []types.Cluster{
				{
					ClusterArn:   aws.String(testArn1),
					ClusterName:  aws.String(testName1),
					CreationTime: aws.Time(now),
					State:        types.ClusterStateActive,
				},
				{
					ClusterArn:   aws.String(testArn2),
					ClusterName:  aws.String(testName2),
					CreationTime: aws.Time(now.Add(1 * time.Hour)),
					State:        types.ClusterStateActive,
				},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testArn1, testArn2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile("test-cluster-1"),
					}},
				},
			},
			expected: []string{testArn2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{testArn1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listMSKClusters(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestMSKCluster_ShouldInclude(t *testing.T) {
	t.Parallel()

	clusterName := "test-cluster"
	now := time.Now()

	tests := map[string]struct {
		cluster  types.Cluster
		expected bool
	}{
		"active cluster": {
			cluster: types.Cluster{
				ClusterName:  &clusterName,
				State:        types.ClusterStateActive,
				CreationTime: &now,
			},
			expected: true,
		},
		"deleting cluster": {
			cluster: types.Cluster{
				ClusterName:  &clusterName,
				State:        types.ClusterStateDeleting,
				CreationTime: &now,
			},
			expected: false,
		},
		"creating cluster": {
			cluster: types.Cluster{
				ClusterName:  &clusterName,
				State:        types.ClusterStateCreating,
				CreationTime: &now,
			},
			expected: false,
		},
		"maintenance cluster": {
			cluster: types.Cluster{
				ClusterName:  &clusterName,
				State:        types.ClusterStateMaintenance,
				CreationTime: &now,
			},
			expected: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := shouldIncludeMSKCluster(tc.cluster, config.ResourceType{})
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestMSKCluster_Delete(t *testing.T) {
	t.Parallel()

	mock := &mockMSKClient{}
	err := deleteMSKCluster(context.Background(), mock, aws.String("test-arn"))
	require.NoError(t, err)
}
