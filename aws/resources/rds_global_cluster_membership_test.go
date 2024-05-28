package resources

import (
	"context"
	"regexp"
	"strings"
	"testing"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
)

type mockedDBGlobalClusterMemberships struct {
	rdsiface.RDSAPI
	DescribeGlobalClustersOutput  rds.DescribeGlobalClustersOutput
	DescribeGlobalClustersError   error
	RemoveFromGlobalClusterOutput rds.RemoveFromGlobalClusterOutput
}

func (m mockedDBGlobalClusterMemberships) RemoveFromGlobalClusterWithContext(_ awsgo.Context, _ *rds.RemoveFromGlobalClusterInput, _ ...request.Option) (*rds.RemoveFromGlobalClusterOutput, error) {
	return &m.RemoveFromGlobalClusterOutput, nil
}

func (m mockedDBGlobalClusterMemberships) DescribeGlobalClusters(input *rds.DescribeGlobalClustersInput) (*rds.DescribeGlobalClustersOutput, error) {
	return &m.DescribeGlobalClustersOutput, m.DescribeGlobalClustersError
}

func (m mockedDBGlobalClusterMemberships) DescribeGlobalClustersWithContext(_ awsgo.Context, _ *rds.DescribeGlobalClustersInput, _ ...request.Option) (*rds.DescribeGlobalClustersOutput, error) {
	return &m.DescribeGlobalClustersOutput, m.DescribeGlobalClustersError
}

func TestRDSGlobalClusterMembershipGetAll(t *testing.T) {
	t.Parallel()

	testName := "test-db-global-cluster"
	dbCluster := DBGlobalClusterMemberships{
		Client: mockedDBGlobalClusterMemberships{
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
	clusters, err := dbCluster.getAll(context.Background(), config.Config{DBGlobalClusterMemberships: config.ResourceType{}})
	assert.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(clusters), strings.ToLower(testName))

	// Testing db cluster exclusion
	clusters, err = dbCluster.getAll(context.Background(), config.Config{
		DBGlobalClusterMemberships: config.ResourceType{
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

func TestRDSGlobalClusterMembershipNukeAll(t *testing.T) {

	t.Parallel()

	testName := "test-db-global-cluster"
	dbCluster := DBGlobalClusterMemberships{
		Client: mockedDBGlobalClusterMemberships{
			DescribeGlobalClustersOutput: rds.DescribeGlobalClustersOutput{
				GlobalClusters: []*rds.GlobalCluster{
					{
						GlobalClusterIdentifier: &testName,
						GlobalClusterMembers:    []*rds.GlobalClusterMember{},
					},
				},
			},
			RemoveFromGlobalClusterOutput: rds.RemoveFromGlobalClusterOutput{},
		},
	}

	err := dbCluster.nukeAll([]*string{&testName})
	assert.NoError(t, err)
}
