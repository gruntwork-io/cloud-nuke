package resources

import (
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/kafka/types"
	"github.com/gruntwork-io/cloud-nuke/config"
)

func TestShouldIncludeMSKCluster(t *testing.T) {
	clusterName := "test-cluster"
	creationTime := time.Now()

	tests := map[string]struct {
		clusterInfo  types.Cluster
		excludeAfter time.Time
		configObj    config.Config
		expected     bool
	}{
		"cluster is in deleting state": {
			clusterInfo: types.Cluster{
				ClusterName:  &clusterName,
				State:        types.ClusterStateDeleting,
				CreationTime: &creationTime,
			},
			excludeAfter: time.Now(),
			configObj:    config.Config{},
			expected:     false,
		},
		"cluster is in creating state": {
			clusterInfo: types.Cluster{
				ClusterName:  &clusterName,
				State:        types.ClusterStateCreating,
				CreationTime: &creationTime,
			},
			excludeAfter: time.Now(),
			configObj:    config.Config{},
			expected:     false,
		},
		"cluster is in active state": {
			clusterInfo: types.Cluster{
				ClusterName:  &clusterName,
				State:        types.ClusterStateActive,
				CreationTime: &creationTime,
			},
			excludeAfter: time.Now(),
			configObj:    config.Config{},
			expected:     true,
		},
		"cluster created before excludeAfter": {
			clusterInfo: types.Cluster{
				ClusterName:  &clusterName,
				State:        types.ClusterStateActive,
				CreationTime: &creationTime,
			},
			excludeAfter: time.Now().Add(1 * time.Hour),
			configObj:    config.Config{},
			expected:     true,
		},
		"cluster created after excludeAfter": {
			clusterInfo: types.Cluster{
				ClusterName:  &clusterName,
				State:        types.ClusterStateActive,
				CreationTime: &creationTime,
			},
			excludeAfter: time.Now().Add(-1 * time.Hour),
			configObj:    config.Config{},
			expected:     false,
		},
		"cluster excluded by name": {
			clusterInfo: types.Cluster{
				ClusterName:  &clusterName,
				State:        types.ClusterStateActive,
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
			clusterInfo: types.Cluster{
				ClusterName:  &clusterName,
				State:        types.ClusterStateActive,
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
