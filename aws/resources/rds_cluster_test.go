package resources

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDBClustersClient struct {
	DescribeDBClustersOutput rds.DescribeDBClustersOutput
	DescribeDBClustersError  error
	DeleteDBClusterOutput    rds.DeleteDBClusterOutput
	DeleteDBClusterError     error
	ModifyDBClusterOutput    rds.ModifyDBClusterOutput
	ModifyDBClusterError     error
}

func (m *mockDBClustersClient) DeleteDBCluster(ctx context.Context, params *rds.DeleteDBClusterInput, optFns ...func(*rds.Options)) (*rds.DeleteDBClusterOutput, error) {
	return &m.DeleteDBClusterOutput, m.DeleteDBClusterError
}

func (m *mockDBClustersClient) DescribeDBClusters(ctx context.Context, params *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error) {
	return &m.DescribeDBClustersOutput, m.DescribeDBClustersError
}

func (m *mockDBClustersClient) ModifyDBCluster(ctx context.Context, params *rds.ModifyDBClusterInput, optFns ...func(*rds.Options)) (*rds.ModifyDBClusterOutput, error) {
	return &m.ModifyDBClusterOutput, m.ModifyDBClusterError
}

func TestDBClusters_ResourceName(t *testing.T) {
	t.Parallel()
	r := NewDBClusters()
	assert.Equal(t, "rds-cluster", r.ResourceName())
}

func TestDBClusters_MaxBatchSize(t *testing.T) {
	t.Parallel()
	r := NewDBClusters()
	assert.Equal(t, DefaultBatchSize, r.MaxBatchSize())
}

func TestListDBClusters(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tests := []struct {
		name     string
		clusters []types.DBCluster
		cfg      config.ResourceType
		expected []string
	}{
		{
			name: "returns all clusters when no filter",
			clusters: []types.DBCluster{
				{DBClusterIdentifier: aws.String("cluster-1"), ClusterCreateTime: &now},
				{DBClusterIdentifier: aws.String("cluster-2"), ClusterCreateTime: &now},
			},
			cfg:      config.ResourceType{},
			expected: []string{"cluster-1", "cluster-2"},
		},
		{
			name: "filters by time exclusion",
			clusters: []types.DBCluster{
				{DBClusterIdentifier: aws.String("old-cluster"), ClusterCreateTime: aws.Time(now.Add(-1 * time.Hour))},
				{DBClusterIdentifier: aws.String("new-cluster"), ClusterCreateTime: &now},
			},
			cfg: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-30 * time.Minute)),
				},
			},
			expected: []string{"old-cluster"},
		},
		{
			name: "filters by name pattern",
			clusters: []types.DBCluster{
				{DBClusterIdentifier: aws.String("prod-cluster"), ClusterCreateTime: &now},
				{DBClusterIdentifier: aws.String("dev-cluster"), ClusterCreateTime: &now},
			},
			cfg: config.ResourceType{
				IncludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("^dev-")}},
				},
			},
			expected: []string{"dev-cluster"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &mockDBClustersClient{
				DescribeDBClustersOutput: rds.DescribeDBClustersOutput{
					DBClusters: tt.clusters,
				},
			}

			clusters, err := listDBClusters(context.Background(), mock, resource.Scope{}, tt.cfg)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, aws.ToStringSlice(clusters))
		})
	}
}

func TestDeleteDBCluster(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		mock        *mockDBClustersClient
		expectError bool
	}{
		{
			name: "successfully deletes cluster",
			mock: &mockDBClustersClient{
				ModifyDBClusterOutput: rds.ModifyDBClusterOutput{},
				DeleteDBClusterOutput: rds.DeleteDBClusterOutput{},
			},
			expectError: false,
		},
		{
			name: "continues on modify error but returns delete error",
			mock: &mockDBClustersClient{
				ModifyDBClusterError:  errors.New("modify failed"),
				DeleteDBClusterOutput: rds.DeleteDBClusterOutput{},
			},
			expectError: false,
		},
		{
			name: "returns error when delete fails",
			mock: &mockDBClustersClient{
				ModifyDBClusterOutput: rds.ModifyDBClusterOutput{},
				DeleteDBClusterError:  errors.New("delete failed"),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := deleteDBCluster(context.Background(), tt.mock, aws.String("test-cluster"))
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
