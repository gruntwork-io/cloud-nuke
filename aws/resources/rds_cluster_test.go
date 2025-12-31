package resources

import (
	"context"
	"strings"
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
	ModifyDBClusterOutput    rds.ModifyDBClusterOutput
}

func (m *mockDBClustersClient) DeleteDBCluster(ctx context.Context, params *rds.DeleteDBClusterInput, optFns ...func(*rds.Options)) (*rds.DeleteDBClusterOutput, error) {
	return &m.DeleteDBClusterOutput, nil
}

func (m *mockDBClustersClient) DescribeDBClusters(ctx context.Context, params *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error) {
	return &m.DescribeDBClustersOutput, m.DescribeDBClustersError
}

func (m *mockDBClustersClient) ModifyDBCluster(ctx context.Context, params *rds.ModifyDBClusterInput, optFns ...func(*rds.Options)) (*rds.ModifyDBClusterOutput, error) {
	return &m.ModifyDBClusterOutput, nil
}

func TestDBClusters_ResourceName(t *testing.T) {
	r := NewDBClusters()
	assert.Equal(t, "rds-cluster", r.ResourceName())
}

func TestDBClusters_MaxBatchSize(t *testing.T) {
	r := NewDBClusters()
	assert.Equal(t, 49, r.MaxBatchSize())
}

func TestListDBClusters(t *testing.T) {
	t.Parallel()

	testName := "test-db-cluster"
	testProtectedName := "test-protected-cluster"
	now := time.Now()

	mock := &mockDBClustersClient{
		DescribeDBClustersOutput: rds.DescribeDBClustersOutput{
			DBClusters: []types.DBCluster{
				{
					DBClusterIdentifier: &testName,
					ClusterCreateTime:   &now,
					DeletionProtection:  aws.Bool(false),
				},
				{
					DBClusterIdentifier: &testProtectedName,
					ClusterCreateTime:   &now,
					DeletionProtection:  aws.Bool(true),
				},
			},
		},
	}

	// Test Case 1: Empty config - should include both protected and unprotected clusters
	clusters, err := listDBClusters(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	assert.Contains(t, aws.ToStringSlice(clusters), strings.ToLower(testName))
	assert.Contains(t, aws.ToStringSlice(clusters), strings.ToLower(testProtectedName))
}

func TestListDBClusters_WithTimeFilter(t *testing.T) {
	t.Parallel()

	testName := "test-db-cluster"
	testProtectedName := "test-protected-cluster"
	now := time.Now()

	mock := &mockDBClustersClient{
		DescribeDBClustersOutput: rds.DescribeDBClustersOutput{
			DBClusters: []types.DBCluster{
				{
					DBClusterIdentifier: &testName,
					ClusterCreateTime:   &now,
					DeletionProtection:  aws.Bool(false),
				},
				{
					DBClusterIdentifier: &testProtectedName,
					ClusterCreateTime:   &now,
					DeletionProtection:  aws.Bool(true),
				},
			},
		},
	}

	// Time-based exclusion - should exclude all clusters created after specified time
	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			TimeAfter: aws.Time(now.Add(-1)),
		},
	}

	clusters, err := listDBClusters(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	assert.NotContains(t, aws.ToStringSlice(clusters), strings.ToLower(testName))
	assert.NotContains(t, aws.ToStringSlice(clusters), strings.ToLower(testProtectedName))
}

func TestDeleteDBCluster(t *testing.T) {
	t.Parallel()

	testName := "test-db-cluster"
	mock := &mockDBClustersClient{
		DescribeDBClustersOutput: rds.DescribeDBClustersOutput{},
		DescribeDBClustersError:  &types.DBClusterNotFoundFault{},
		ModifyDBClusterOutput:    rds.ModifyDBClusterOutput{},
		DeleteDBClusterOutput:    rds.DeleteDBClusterOutput{},
	}

	err := deleteDBCluster(context.Background(), mock, aws.String(testName))
	require.NoError(t, err)
}
