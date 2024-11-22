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

type mockedDBGlobalClusterMemberships struct {
	DBGCMembershipsAPI
	DescribeGlobalClustersOutput  rds.DescribeGlobalClustersOutput
	DescribeGlobalClustersError   error
	RemoveFromGlobalClusterOutput rds.RemoveFromGlobalClusterOutput
}

func (m mockedDBGlobalClusterMemberships) RemoveFromGlobalCluster(ctx context.Context, params *rds.RemoveFromGlobalClusterInput, optFns ...func(*rds.Options)) (*rds.RemoveFromGlobalClusterOutput, error) {
	return &m.RemoveFromGlobalClusterOutput, nil
}

func (m mockedDBGlobalClusterMemberships) DescribeGlobalClusters(ctx context.Context, params *rds.DescribeGlobalClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeGlobalClustersOutput, error) {
	return &m.DescribeGlobalClustersOutput, m.DescribeGlobalClustersError
}

func TestRDSGlobalClusterMembershipGetAll(t *testing.T) {
	t.Parallel()

	testName := "test-db-global-cluster"
	dbCluster := DBGlobalClusterMemberships{
		Client: mockedDBGlobalClusterMemberships{
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
	clusters, err := dbCluster.getAll(context.Background(), config.Config{DBGlobalClusterMemberships: config.ResourceType{}})
	assert.NoError(t, err)
	assert.Contains(t, aws.ToStringSlice(clusters), strings.ToLower(testName))

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
	assert.NotContains(t, aws.ToStringSlice(clusters), strings.ToLower(testName))
}

func TestRDSGlobalClusterMembershipNukeAll(t *testing.T) {

	t.Parallel()

	testName := "test-db-global-cluster"
	dbCluster := DBGlobalClusterMemberships{
		Client: mockedDBGlobalClusterMemberships{
			DescribeGlobalClustersOutput: rds.DescribeGlobalClustersOutput{
				GlobalClusters: []types.GlobalCluster{
					{
						GlobalClusterIdentifier: &testName,
						GlobalClusterMembers:    []types.GlobalClusterMember{},
					},
				},
			},
			RemoveFromGlobalClusterOutput: rds.RemoveFromGlobalClusterOutput{},
		},
	}

	err := dbCluster.nukeAll([]*string{&testName})
	assert.NoError(t, err)
}
