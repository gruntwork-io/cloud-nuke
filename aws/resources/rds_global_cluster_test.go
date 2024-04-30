package resources

import (
	"context"
	"regexp"
	"strings"
	"testing"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
)

type mockedDBGlobalClusters struct {
	rdsiface.RDSAPI
	DescribeGlobalClustersOutput rds.DescribeGlobalClustersOutput
	DescribeGlobalClustersError  error
	DeleteGlobalClusterOutput    rds.DeleteGlobalClusterOutput
}

func (m mockedDBGlobalClusters) DeleteGlobalCluster(input *rds.DeleteGlobalClusterInput) (*rds.DeleteGlobalClusterOutput, error) {
	return &m.DeleteGlobalClusterOutput, nil
}

func (m mockedDBGlobalClusters) DescribeGlobalClusters(input *rds.DescribeGlobalClustersInput) (*rds.DescribeGlobalClustersOutput, error) {
	return &m.DescribeGlobalClustersOutput, m.DescribeGlobalClustersError
}

func (m mockedDBGlobalClusters) DescribeGlobalClustersWithContext(ctx context.Context, input *rds.DescribeGlobalClustersInput, _ ...request.Option) (*rds.DescribeGlobalClustersOutput, error) {
	return &m.DescribeGlobalClustersOutput, m.DescribeGlobalClustersError
}

func TestRDSGlobalClusterGetAll(t *testing.T) {
	t.Parallel()

	testName := "test-db-global-cluster"
	dbCluster := DBGlobalClusters{
		Client: mockedDBGlobalClusters{
			DescribeGlobalClustersOutput: rds.DescribeGlobalClustersOutput{
				GlobalClusters: []*rds.GlobalCluster{
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
	assert.Contains(t, awsgo.StringValueSlice(clusters), strings.ToLower(testName))

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
	assert.NotContains(t, awsgo.StringValueSlice(clusters), strings.ToLower(testName))
}

func TestRDSGlobalClusterNukeAll(t *testing.T) {

	t.Parallel()

	testName := "test-db-global-cluster"
	dbCluster := DBGlobalClusters{
		Client: mockedDBGlobalClusters{
			DescribeGlobalClustersOutput: rds.DescribeGlobalClustersOutput{},
			DescribeGlobalClustersError:  awserr.New(rds.ErrCodeGlobalClusterNotFoundFault, "", nil),
			DeleteGlobalClusterOutput:    rds.DeleteGlobalClusterOutput{},
		},
	}

	err := dbCluster.nukeAll([]*string{&testName})
	assert.NoError(t, err)
}
