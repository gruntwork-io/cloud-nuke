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
	"github.com/stretchr/testify/assert"
)

type mockedDBClusters struct {
	DBClustersAPI
	DescribeDBClustersOutput rds.DescribeDBClustersOutput
	DescribeDBClustersError  error
	DeleteDBClusterOutput    rds.DeleteDBClusterOutput
	ModifyDBClusterOutput    rds.ModifyDBClusterOutput
}

func (m mockedDBClusters) waitUntilRdsClusterDeleted(*rds.DescribeDBClustersInput) error {
	return nil
}

func (m mockedDBClusters) DeleteDBCluster(ctx context.Context, params *rds.DeleteDBClusterInput, optFns ...func(*rds.Options)) (*rds.DeleteDBClusterOutput, error) {
	return &m.DeleteDBClusterOutput, nil
}

func (m mockedDBClusters) DescribeDBClusters(ctx context.Context, params *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error) {
	return &m.DescribeDBClustersOutput, m.DescribeDBClustersError
}

func (m mockedDBClusters) ModifyDBCluster(ctx context.Context, params *rds.ModifyDBClusterInput, optFns ...func(*rds.Options)) (*rds.ModifyDBClusterOutput, error) {
	return &m.ModifyDBClusterOutput, nil
}

func TestRDSClusterGetAll(t *testing.T) {
	t.Parallel()

	// Test data setup
	testName := "test-db-cluster"
	testProtectedName := "test-protected-cluster"
	now := time.Now()
	dbCluster := DBClusters{
		Client: mockedDBClusters{
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
		},
	}

	// Test Case 1: Empty config - should exclude deletion-protected clusters by default
	clusters, err := dbCluster.getAll(context.Background(), config.Config{DBClusters: config.AWSProtectectableResourceType{}})
	assert.NoError(t, err)
	assert.Contains(t, aws.ToStringSlice(clusters), strings.ToLower(testName))
	assert.NotContains(t, aws.ToStringSlice(clusters), strings.ToLower(testProtectedName))

	// Test Case 2: IncludeDeletionProtected=true - should include both protected and unprotected clusters
	clusters, err = dbCluster.getAll(context.Background(), config.Config{
		DBClusters: config.AWSProtectectableResourceType{
			IncludeDeletionProtected: true,
		},
	})
	assert.NoError(t, err)
	assert.Contains(t, aws.ToStringSlice(clusters), strings.ToLower(testName))
	assert.Contains(t, aws.ToStringSlice(clusters), strings.ToLower(testProtectedName))

	// Test Case 3: Time-based exclusion - should exclude all clusters created after specified time
	clusters, err = dbCluster.getAll(context.Background(), config.Config{
		DBClusters: config.AWSProtectectableResourceType{
			ResourceType: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1)),
				},
			},
		},
	})
	assert.NoError(t, err)
	assert.NotContains(t, aws.ToStringSlice(clusters), strings.ToLower(testName))
	assert.NotContains(t, aws.ToStringSlice(clusters), strings.ToLower(testProtectedName))
}

func TestRDSClusterNukeAll(t *testing.T) {
	t.Parallel()

	testName := "test-db-cluster"
	dbCluster := DBClusters{
		Client: mockedDBClusters{
			DescribeDBClustersOutput: rds.DescribeDBClustersOutput{},
			DescribeDBClustersError:  &types.DBClusterNotFoundFault{},
			ModifyDBClusterOutput:    rds.ModifyDBClusterOutput{},
			DeleteDBClusterOutput:    rds.DeleteDBClusterOutput{},
		},
	}

	err := dbCluster.nukeAll([]*string{&testName})
	assert.NoError(t, err)
}
