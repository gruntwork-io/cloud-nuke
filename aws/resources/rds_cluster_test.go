package resources

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/andrewderr/cloud-nuke-a1/config"
	"github.com/andrewderr/cloud-nuke-a1/telemetry"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/stretchr/testify/assert"
)

type mockedDBClusters struct {
	rdsiface.RDSAPI
	DescribeDBClustersOutput rds.DescribeDBClustersOutput
	DescribeDBClustersError  error
	DeleteDBClusterOutput    rds.DeleteDBClusterOutput
}

func (m mockedDBClusters) waitUntilRdsClusterDeleted(*rds.DescribeDBClustersInput) error {
	return nil
}

func (m mockedDBClusters) DeleteDBCluster(input *rds.DeleteDBClusterInput) (*rds.DeleteDBClusterOutput, error) {
	return &m.DeleteDBClusterOutput, nil
}

func (m mockedDBClusters) DescribeDBClusters(input *rds.DescribeDBClustersInput) (*rds.DescribeDBClustersOutput, error) {
	return &m.DescribeDBClustersOutput, m.DescribeDBClustersError
}

func TestRDSClusterGetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName := "test-db-cluster"
	now := time.Now()
	dbCluster := DBClusters{
		Client: mockedDBClusters{
			DescribeDBClustersOutput: rds.DescribeDBClustersOutput{
				DBClusters: []*rds.DBCluster{{
					DBClusterIdentifier: &testName,
					ClusterCreateTime:   &now,
				}},
			},
		},
	}

	// Testing empty config
	clusters, err := dbCluster.getAll(context.Background(), config.Config{DBClusters: config.ResourceType{}})
	assert.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(clusters), strings.ToLower(testName))

	// Testing db cluster exclusion
	clusters, err = dbCluster.getAll(context.Background(), config.Config{
		DBClusters: config.ResourceType{
			ExcludeRule: config.FilterRule{
				TimeAfter: awsgo.Time(now.Add(-1)),
			},
		},
	})
	assert.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(clusters), strings.ToLower(testName))
}

func TestRDSClusterNukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName := "test-db-cluster"
	dbCluster := DBClusters{
		Client: mockedDBClusters{
			DescribeDBClustersOutput: rds.DescribeDBClustersOutput{},
			DescribeDBClustersError:  awserr.New(rds.ErrCodeDBClusterNotFoundFault, "", nil),
			DeleteDBClusterOutput:    rds.DeleteDBClusterOutput{},
		},
	}

	err := dbCluster.nukeAll([]*string{&testName})
	assert.NoError(t, err)
}
