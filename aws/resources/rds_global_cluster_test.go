package resources

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
)

type mockedDBGlobalClusters struct {
	DBGlobalClustersAPI
	DescribeGlobalClustersOutput rds.DescribeGlobalClustersOutput
	DescribeGlobalClustersError  error
	DeleteGlobalClusterOutput    rds.DeleteGlobalClusterOutput
}

func (m mockedDBGlobalClusters) DeleteGlobalCluster(ctx context.Context, params *rds.DeleteGlobalClusterInput, optFns ...func(*rds.Options)) (*rds.DeleteGlobalClusterOutput, error) {
	return &m.DeleteGlobalClusterOutput, nil
}

func (m mockedDBGlobalClusters) DescribeGlobalClusters(ctx context.Context, input *rds.DescribeGlobalClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeGlobalClustersOutput, error) {
	return &m.DescribeGlobalClustersOutput, m.DescribeGlobalClustersError
}

func TestRDSGlobalClusterGetAll(t *testing.T) {
	t.Parallel()

	testName := "test-db-global-cluster"
	dbCluster := DBGlobalClusters{
		Client: mockedDBGlobalClusters{
			DescribeGlobalClustersOutput: rds.DescribeGlobalClustersOutput{
				GlobalClusters: []types.GlobalCluster{
					{
						GlobalClusterIdentifier: &testName,
					},
				},
			},
		},
	}

	// Testing empty config
	clusters, err := dbCluster.getAll(context.Background(), config.Config{DBGlobalClusters: config.ResourceType{}})
	assert.NoError(t, err)
	assert.Contains(t, aws.ToStringSlice(clusters), strings.ToLower(testName))

	// Testing db cluster exclusion
	clusters, err = dbCluster.getAll(context.Background(), config.Config{
		DBGlobalClusters: config.ResourceType{
			ExcludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{{
					RE: *regexp.MustCompile(testName),
				}},
			},
		},
	})
	assert.NoError(t, err)
	assert.NotContains(t, aws.ToStringSlice(clusters), strings.ToLower(testName))
}

func TestRDSGlobalClusterNukeAll(t *testing.T) {
	t.Parallel()

	testName := "test-db-global-cluster"
	dbCluster := DBGlobalClusters{
		Client: mockedDBGlobalClusters{
			DescribeGlobalClustersOutput: rds.DescribeGlobalClustersOutput{},
			DescribeGlobalClustersError:  &types.GlobalClusterNotFoundFault{},
			DeleteGlobalClusterOutput:    rds.DeleteGlobalClusterOutput{},
		},
	}

	err := dbCluster.nukeAll([]*string{aws.String(testName)})
	assert.NoError(t, err)
}
