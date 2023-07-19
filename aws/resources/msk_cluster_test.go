package resources

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	kafkatypes "github.com/aws/aws-sdk-go-v2/service/kafka/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/require"
)

func deferMSKTerminateWithTestResult(t *testing.T, cluster *TestMSKCluster) {
	if err := cluster.terminate(context.Background()); err != nil {
		t.Fatalf("failed to terminate test MSK cluster: %v", err)
	}
}

func TestGetAllMSKClusters(t *testing.T) {
	ctx := context.Background()

	region, err := getRandomRegion()
	if err != nil {
		t.Fatalf("failed to get random region: %v", err)
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		t.Fatalf("failed to load AWS config: %v", err)
	}

	clusterName := fmt.Sprintf("cloud-nuke-msk-test-%s", random.UniqueId())
	cluster, err := createTestMSKCluster(ctx, cfg, clusterName, 1)
	if err != nil {
		t.Fatalf("failed to create test MSK cluster: %v", err)
	}
	defer deferMSKTerminateWithTestResult(t, &cluster)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)

	// test that we can find the cluster
	clusterInfo, err := getAllMSKClusters(session, time.Now(), config.Config{})
	if err != nil {
		t.Fatalf("failed to get all MSK clusters: %v", err)
	}

	if len(clusterInfo) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusterInfo))
	}

	// test that we can then delete the cluster
	err = nukeAllMSKClusters(session, clusterInfo)
	if err != nil {
		t.Fatalf("failed to nuke MSK clusters: %v", err)
	}

	// test that we can't find the cluster anymore
	clusterInfo, err = getAllMSKClusters(session, time.Now(), config.Config{})
	if err != nil {
		t.Fatalf("failed to get all MSK clusters: %v", err)
	}

	if len(clusterInfo) != 0 {
		t.Fatalf("expected 0 clusters, got %d", len(clusterInfo))
	}
}

func TestShouldIncludeMSKCluster(t *testing.T) {
	clusterName := "test-cluster"
	creationTime := time.Now()

	tests := map[string]struct {
		clusterInfo  kafkatypes.Cluster
		excludeAfter time.Time
		configObj    config.Config
		expected     bool
	}{
		"cluster is in deleting state": {
			clusterInfo: kafkatypes.Cluster{
				ClusterName:  &clusterName,
				State:        kafkatypes.ClusterStateDeleting,
				CreationTime: &creationTime,
			},
			excludeAfter: time.Now(),
			configObj:    config.Config{},
			expected:     false,
		},
		"cluster is in creating state": {
			clusterInfo: kafkatypes.Cluster{
				ClusterName:  &clusterName,
				State:        kafkatypes.ClusterStateCreating,
				CreationTime: &creationTime,
			},
			excludeAfter: time.Now(),
			configObj:    config.Config{},
			expected:     false,
		},
		"cluster is in active state": {
			clusterInfo: kafkatypes.Cluster{
				ClusterName:  &clusterName,
				State:        kafkatypes.ClusterStateActive,
				CreationTime: &creationTime,
			},
			excludeAfter: time.Now(),
			configObj:    config.Config{},
			expected:     true,
		},
		"cluster created before excludeAfter": {
			clusterInfo: kafkatypes.Cluster{
				ClusterName:  &clusterName,
				State:        kafkatypes.ClusterStateActive,
				CreationTime: &creationTime,
			},
			excludeAfter: time.Now().Add(1 * time.Hour),
			configObj:    config.Config{},
			expected:     true,
		},
		"cluster created after excludeAfter": {
			clusterInfo: kafkatypes.Cluster{
				ClusterName:  &clusterName,
				State:        kafkatypes.ClusterStateActive,
				CreationTime: &creationTime,
			},
			excludeAfter: time.Now().Add(-1 * time.Hour),
			configObj:    config.Config{},
			expected:     false,
		},
		"cluster excluded by name": {
			clusterInfo: kafkatypes.Cluster{
				ClusterName:  &clusterName,
				State:        kafkatypes.ClusterStateActive,
				CreationTime: &creationTime,
			},
			excludeAfter: time.Now(),
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
			clusterInfo: kafkatypes.Cluster{
				ClusterName:  &clusterName,
				State:        kafkatypes.ClusterStateActive,
				CreationTime: &creationTime,
			},
			excludeAfter: time.Now(),
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
			actual := shouldIncludeMSKCluster(tc.clusterInfo, tc.excludeAfter, tc.configObj)
			if actual != tc.expected {
				t.Fatalf("Expected %v, got %v", tc.expected, actual)
			}
		})
	}
}
