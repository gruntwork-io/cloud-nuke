package resources

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kafka"
	"github.com/aws/aws-sdk-go/service/kafka/kafkaiface"
	"github.com/gruntwork-io/cloud-nuke/config"
)

type mockMSKClient struct {
	kafkaiface.KafkaAPI
	listClustersV2PagesFn func(input *kafka.ListClustersV2Input, callback func(*kafka.ListClustersV2Output, bool) bool) error
	deleteClusterFn       func(input *kafka.DeleteClusterInput) (*kafka.DeleteClusterOutput, error)
}

func (m mockMSKClient) ListClustersV2Pages(input *kafka.ListClustersV2Input, callback func(*kafka.ListClustersV2Output, bool) bool) error {
	return m.listClustersV2PagesFn(input, callback)
}

func (m mockMSKClient) DeleteCluster(input *kafka.DeleteClusterInput) (*kafka.DeleteClusterOutput, error) {
	return nil, nil
}

func TestListMSKClustersSingle(t *testing.T) {
	mockMskClient := mockMSKClient{
		listClustersV2PagesFn: func(input *kafka.ListClustersV2Input, callback func(*kafka.ListClustersV2Output, bool) bool) error {
			callback(&kafka.ListClustersV2Output{
				ClusterInfoList: []*kafka.Cluster{
					{
						ClusterArn:   aws.String("arn:aws:kafka:us-east-1:123456789012:cluster/test-cluster-1/1a2b3c4d-5e6f-7g8h-9i0j-1k2l3m4n5o6p"),
						ClusterName:  aws.String("test-cluster-1"),
						CreationTime: aws.Time(time.Now()),
						State:        aws.String(kafka.ClusterStateActive),
					},
				},
			}, true)
			return nil
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
		listClustersV2PagesFn: func(input *kafka.ListClustersV2Input, callback func(*kafka.ListClustersV2Output, bool) bool) error {
			callback(&kafka.ListClustersV2Output{
				ClusterInfoList: []*kafka.Cluster{
					{
						ClusterArn:   aws.String("arn:aws:kafka:us-east-1:123456789012:cluster/test-cluster-1/1a2b3c4d-5e6f-7g8h-9i0j-1k2l3m4n5o6p"),
						ClusterName:  aws.String("test-cluster-1"),
						CreationTime: aws.Time(time.Now()),
						State:        aws.String(kafka.ClusterStateActive),
					}, {
						ClusterArn:   aws.String("arn:aws:kafka:us-east-1:123456789012:cluster/test-cluster-2/1a2b3c4d-5e6f-7g8h-9i0j-1k2l3m4n5o6p"),
						ClusterName:  aws.String("test-cluster-2"),
						CreationTime: aws.Time(time.Now()),
						State:        aws.String(kafka.ClusterStateActive),
					}, {
						ClusterArn:   aws.String("arn:aws:kafka:us-east-1:123456789012:cluster/test-cluster-3/1a2b3c4d-5e6f-7g8h-9i0j-1k2l3m4n5o6p"),
						ClusterName:  aws.String("test-cluster-3"),
						CreationTime: aws.Time(time.Now()),
						State:        aws.String(kafka.ClusterStateActive),
					},
				},
			}, true)
			return nil
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

func TestGetAllMSKError(t *testing.T) {
	mockMskClient := mockMSKClient{
		listClustersV2PagesFn: func(input *kafka.ListClustersV2Input, callback func(*kafka.ListClustersV2Output, bool) bool) error {
			return fmt.Errorf("Error listing MSK Clusters")
		},
	}

	msk := MSKCluster{
		Client: &mockMskClient,
	}

	_, err := msk.getAll(context.Background(), config.Config{})
	if err == nil {
		t.Fatalf("Expected error listing MSK Clusters")
	}
}

func TestShouldIncludeMSKCluster(t *testing.T) {
	clusterName := "test-cluster"
	creationTime := time.Now()

	tests := map[string]struct {
		cluster   kafka.Cluster
		configObj config.Config
		expected  bool
	}{
		"cluster is in deleting state": {
			cluster: kafka.Cluster{
				ClusterName:  &clusterName,
				State:        aws.String(kafka.ClusterStateDeleting),
				CreationTime: &creationTime,
			},
			configObj: config.Config{},
			expected:  false,
		},
		"cluster is in creating state": {
			cluster: kafka.Cluster{
				ClusterName:  &clusterName,
				State:        aws.String(kafka.ClusterStateCreating),
				CreationTime: &creationTime,
			},
			configObj: config.Config{},
			expected:  false,
		},
		"cluster is in active state": {
			cluster: kafka.Cluster{
				ClusterName:  &clusterName,
				State:        aws.String(kafka.ClusterStateActive),
				CreationTime: &creationTime,
			},
			configObj: config.Config{},
			expected:  true,
		},
		"cluster excluded by name": {
			cluster: kafka.Cluster{
				ClusterName:  &clusterName,
				State:        aws.String(kafka.ClusterStateActive),
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
			cluster: kafka.Cluster{
				ClusterName:  &clusterName,
				State:        aws.String(kafka.ClusterStateActive),
				CreationTime: &creationTime,
			},
			configObj: config.Config{
				MSKCluster: config.ResourceType{
					IncludeRule: config.FilterRule{
						NamesRegExp: []config.Expression{
							{
								RE: *regexp.MustCompile("test-cluster"),
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
			actual := msk.shouldInclude(&tc.cluster, tc.configObj)
			if actual != tc.expected {
				t.Fatalf("Expected %v, got %v", tc.expected, actual)
			}
		})
	}
}

func TestNukeMSKCluster(t *testing.T) {
	mockMskClient := mockMSKClient{
		deleteClusterFn: func(input *kafka.DeleteClusterInput) (*kafka.DeleteClusterOutput, error) {
			return nil, nil
		},
	}

	msk := MSKCluster{
		Client: &mockMskClient,
	}

	err := msk.Nuke(nil, []string{})
	if err != nil {
		t.Fatalf("Unable to nuke MSK Clusters: %v", err)
	}
}
