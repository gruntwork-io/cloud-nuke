package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedDBSubnetGroups struct {
	DBSubnetGroupsAPI
	DescribeDBSubnetGroupsOutput rds.DescribeDBSubnetGroupsOutput
	DescribeDBSubnetGroupError   error
	DeleteDBSubnetGroupOutput    rds.DeleteDBSubnetGroupOutput
}

func (m mockedDBSubnetGroups) DescribeDBSubnetGroups(ctx context.Context, params *rds.DescribeDBSubnetGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSubnetGroupsOutput, error) {
	return &m.DescribeDBSubnetGroupsOutput, m.DescribeDBSubnetGroupError
}

func (m mockedDBSubnetGroups) DeleteDBSubnetGroup(ctx context.Context, params *rds.DeleteDBSubnetGroupInput, optFns ...func(*rds.Options)) (*rds.DeleteDBSubnetGroupOutput, error) {
	return &m.DeleteDBSubnetGroupOutput, nil
}

var dbSubnetGroupNotFoundError = &types.DBSubnetGroupNotFoundFault{}

func TestDBSubnetGroups_GetAll(t *testing.T) {

	t.Parallel()

	testName1 := "test-db-subnet-group1"
	testName2 := "test-db-subnet-group2"
	dsg := DBSubnetGroups{
		Client: mockedDBSubnetGroups{
			DescribeDBSubnetGroupsOutput: rds.DescribeDBSubnetGroupsOutput{
				DBSubnetGroups: []types.DBSubnetGroup{
					{
						DBSubnetGroupName: aws.String(testName1),
					},
					{
						DBSubnetGroupName: aws.String(testName2),
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
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testName2},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := dsg.getAll(context.Background(), config.Config{
				DBSubnetGroups: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}

}

func TestDBSubnetGroups_NukeAll(t *testing.T) {

	t.Parallel()

	dsg := DBSubnetGroups{
		Client: mockedDBSubnetGroups{
			DeleteDBSubnetGroupOutput:  rds.DeleteDBSubnetGroupOutput{},
			DescribeDBSubnetGroupError: dbSubnetGroupNotFoundError,
		},
	}

	err := dsg.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
