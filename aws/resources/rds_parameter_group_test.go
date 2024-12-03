package resources

import (
	"context"
	"regexp"
	"testing"

	awsgo "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedRdsDBParameterGroup struct {
	RdsParameterGroupAPI
	DescribeDBParameterGroupsOutput rds.DescribeDBParameterGroupsOutput
	DeleteDBParameterGroupOutput    rds.DeleteDBParameterGroupOutput
}

func (m mockedRdsDBParameterGroup) DescribeDBParameterGroups(ctx context.Context, params *rds.DescribeDBParameterGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBParameterGroupsOutput, error) {
	return &m.DescribeDBParameterGroupsOutput, nil
}

func (m mockedRdsDBParameterGroup) DeleteDBParameterGroup(ctx context.Context, params *rds.DeleteDBParameterGroupInput, optFns ...func(*rds.Options)) (*rds.DeleteDBParameterGroupOutput, error) {
	return &m.DeleteDBParameterGroupOutput, nil
}

func TestRDSparameterGroupGetAll(t *testing.T) {
	t.Parallel()

	testName01 := "test-db-paramater-group-01"
	testName02 := "test-db-paramater-group-02"

	pg := RdsParameterGroup{
		Client: mockedRdsDBParameterGroup{
			DescribeDBParameterGroupsOutput: rds.DescribeDBParameterGroupsOutput{
				DBParameterGroups: []types.DBParameterGroup{
					{
						DBParameterGroupName: awsgo.String(testName01),
					},
					{
						DBParameterGroupName: awsgo.String(testName02),
					},
				},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testName01, testName02},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName01),
					}}},
			},
			expected: []string{testName02},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := pg.getAll(context.Background(), config.Config{
				RdsParameterGroup: tc.configObj,
			})
			require.NoError(t, err)

			require.Equal(t, len(tc.expected), len(names))
			for _, name := range names {
				require.Contains(t, tc.expected, *name)
			}
		})
	}
}

func TestRDSParameterGroupNukeAll(t *testing.T) {
	t.Parallel()

	testName := "test-db-parameter-group"
	dbCluster := RdsParameterGroup{
		Client: mockedRdsDBParameterGroup{
			DeleteDBParameterGroupOutput: rds.DeleteDBParameterGroupOutput{},
		},
	}
	err := dbCluster.nukeAll([]*string{&testName})
	assert.NoError(t, err)
}
