package resources

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/aws/aws-sdk-go-v2/service/kafka/types"
	"github.com/gruntwork-io/cloud-nuke/config"
)

type mockMSKClient struct {
	MSKClusterAPI
	ListClustersV2Output kafka.ListClustersV2Output
	DeleteClusterOutput  kafka.DeleteClusterOutput
}

func (m mockMSKClient) ListClustersV2(ctx context.Context, params *kafka.ListClustersV2Input, optFns ...func(*kafka.Options)) (*kafka.ListClustersV2Output, error) {
	return &m.ListClustersV2Output, nil
}

func (m mockMSKClient) DeleteCluster(ctx context.Context, params *kafka.DeleteClusterInput, optFns ...func(*kafka.Options)) (*kafka.DeleteClusterOutput, error) {
	return &m.DeleteClusterOutput, nil
}

func TestListMSKClustersSingle(t *testing.T) {
	mockMskClient := mockMSKClient{
		ListClustersV2Output: kafka.ListClustersV2Output{
			ClusterInfoList: []types.Cluster{
				{
					ClusterArn:   aws.String("arn:aws:kafka:us-east-1:123456789012:cluster/test-cluster-1/1a2b3c4d-5e6f-7g8h-9i0j-1k2l3m4n5o6p"),
					ClusterName:  aws.String("test-cluster-1"),
					CreationTime: aws.Time(time.Now()),
					State:        types.ClusterStateActive,
				},
			},
		},
	}

	msk := MSKCluster{
		Client: &mockMskClient,
	}

	clusterIDs, err := msk.getAll(context.Background(), config.Config{})
	if err != nil {
		t.Fatalf("Unable to list MSK Clusters: %v", err)
	}

	if len(clusterIDs) != 1 {
		t.Fatalf("Expected 1 cluster, got %d", len(clusterIDs))
	}

	if *clusterIDs[0] != "arn:aws:kafka:us-east-1:123456789012:cluster/test-cluster-1/1a2b3c4d-5e6f-7g8h-9i0j-1k2l3m4n5o6p" {
		t.Fatalf("Unexpected cluster ID: %s", *clusterIDs[0])
	}
}

func TestListMSKClustersMultiple(t *testing.T) {
	mockMskClient := mockMSKClient{
		ListClustersV2Output: kafka.ListClustersV2Output{
			ClusterInfoList: []types.Cluster{
				{
					ClusterArn:   aws.String("arn:aws:kafka:us-east-1:123456789012:cluster/test-cluster-1/1a2b3c4d-5e6f-7g8h-9i0j-1k2l3m4n5o6p"),
					ClusterName:  aws.String("test-cluster-1"),
					CreationTime: aws.Time(time.Now()),
					State:        types.ClusterStateActive,
				}, {
					ClusterArn:   aws.String("arn:aws:kafka:us-east-1:123456789012:cluster/test-cluster-2/1a2b3c4d-5e6f-7g8h-9i0j-1k2l3m4n5o6p"),
					ClusterName:  aws.String("test-cluster-2"),
					CreationTime: aws.Time(time.Now()),
					State:        types.ClusterStateActive,
				}, {
					ClusterArn:   aws.String("arn:aws:kafka:us-east-1:123456789012:cluster/test-cluster-3/1a2b3c4d-5e6f-7g8h-9i0j-1k2l3m4n5o6p"),
					ClusterName:  aws.String("test-cluster-3"),
					CreationTime: aws.Time(time.Now()),
					State:        types.ClusterStateActive,
				},
			},
		},
	}

	msk := MSKCluster{
		Client: &mockMskClient,
	}

	clusterIDs, err := msk.getAll(context.Background(), config.Config{})
	if err != nil {
		t.Fatalf("Unable to list MSK Clusters: %v", err)
	}

	if len(clusterIDs) != 3 {
		t.Fatalf("Expected 3 clusters, got %d", len(clusterIDs))
	}

	for i := range clusterIDs {
		prefix := fmt.Sprintf("arn:aws:kafka:us-east-1:123456789012:cluster/test-cluster-%d", i+1)
		if !strings.HasPrefix(*clusterIDs[i], prefix) {
			t.Fatalf("Unexpected cluster ID: %s", *clusterIDs[i])
		}
	}
}

func TestShouldIncludeMSKCluster(t *testing.T) {
	clusterName := "test-cluster"
	creationTime := time.Now()

	tests := map[string]struct {
		cluster   types.Cluster
		configObj config.Config
		expected  bool
	}{
		"cluster is in deleting state": {
			cluster: types.Cluster{
				ClusterName:  &clusterName,
				State:        types.ClusterStateDeleting,
				CreationTime: &creationTime,
			},
			configObj: config.Config{},
			expected:  false,
		},
		"cluster is in creating state": {
			cluster: types.Cluster{
				ClusterName:  &clusterName,
				State:        types.ClusterStateCreating,
				CreationTime: &creationTime,
			},
			configObj: config.Config{},
			expected:  false,
		},
		"cluster is in active state": {
			cluster: types.Cluster{
				ClusterName:  &clusterName,
				State:        types.ClusterStateActive,
				CreationTime: &creationTime,
			},
			configObj: config.Config{},
			expected:  true,
		},
		"cluster excluded by name": {
			cluster: types.Cluster{
				ClusterName:  &clusterName,
				State:        types.ClusterStateActive,
				CreationTime: &creationTime,
			},
			configObj: config.Config{
				MSKCluster: config.ResourceType{
					ExcludeRule: config.FilterRule{
						NamesRegExp: []config.Expression{
							{
								RE: *regexp.MustCompile("test-cluster"),
							},
						},
					},
				},
			},
			expected: false,
		},
		"cluster included by name": {
			cluster: types.Cluster{
				ClusterName:  &clusterName,
				State:        types.ClusterStateActive,
				CreationTime: &creationTime,
			},
			configObj: config.Config{
				MSKCluster: config.ResourceType{
					IncludeRule: config.FilterRule{
						NamesRegExp: []config.Expression{
							{
								RE: *regexp.MustCompile("^test-cluster"),
							},
						},
					},
				},
			},
			expected: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			msk := MSKCluster{}
			actual := msk.shouldInclude(tc.cluster, tc.configObj)
			if actual != tc.expected {
				t.Fatalf("Expected %v, got %v", tc.expected, actual)
			}
		})
	}
}

func TestNukeMSKCluster(t *testing.T) {
	mockMskClient := mockMSKClient{
		DeleteClusterOutput: kafka.DeleteClusterOutput{},
	}

	msk := MSKCluster{
		Client: &mockMskClient,
	}

	err := msk.Nuke([]string{})
	if err != nil {
		t.Fatalf("Unable to nuke MSK Clusters: %v", err)
	}
}
