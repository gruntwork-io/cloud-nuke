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

func TestRDSClusterGetAll(t *testing.T) {

	t.Parallel()

	testName := "test-db-cluster"
	now := time.Now()
	dbCluster := DBClusters{
		Client: mockedDBClusters{
			DescribeDBClustersOutput: rds.DescribeDBClustersOutput{
				DBClusters: []types.DBCluster{{
					DBClusterIdentifier: &testName,
					ClusterCreateTime:   &now,
				}},
			},
		},
	}

	// Testing empty config
	clusters, err := dbCluster.getAll(context.Background(), config.Config{DBClusters: config.ResourceType{}})
	assert.NoError(t, err)
	assert.Contains(t, aws.ToStringSlice(clusters), strings.ToLower(testName))

	// Testing db cluster exclusion
	clusters, err = dbCluster.getAll(context.Background(), config.Config{
		DBClusters: config.ResourceType{
			ExcludeRule: config.FilterRule{
				TimeAfter: aws.Time(now.Add(-1)),
			},
		},
	})
	assert.NoError(t, err)
	assert.NotContains(t, aws.ToStringSlice(clusters), strings.ToLower(testName))
}

func TestRDSClusterNukeAll(t *testing.T) {
	t.Parallel()

	testName := "test-db-cluster"
	dbCluster := DBClusters{
		Client: mockedDBClusters{
			DescribeDBClustersOutput: rds.DescribeDBClustersOutput{},
			DescribeDBClustersError:  &types.DBClusterNotFoundFault{},
		},
	}

	err := dbCluster.nukeAll([]*string{&testName})
	assert.NoError(t, err)
}
