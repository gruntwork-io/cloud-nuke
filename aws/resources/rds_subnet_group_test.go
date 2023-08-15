package resources

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
)

type mockedDBSubnetGroups struct {
	rdsiface.RDSAPI
	DescribeDBSubnetGroupsOutput rds.DescribeDBSubnetGroupsOutput
	DescribeDBSubnetGroupError   error
	DeleteDBSubnetGroupOutput    rds.DeleteDBSubnetGroupOutput
}

func (m mockedDBSubnetGroups) DescribeDBSubnetGroupsPages(input *rds.DescribeDBSubnetGroupsInput, fn func(*rds.DescribeDBSubnetGroupsOutput, bool) bool) error {
	fn(&m.DescribeDBSubnetGroupsOutput, true)
	return nil
}

func (m mockedDBSubnetGroups) DescribeDBSubnetGroups(*rds.DescribeDBSubnetGroupsInput) (*rds.DescribeDBSubnetGroupsOutput, error) {
	return &m.DescribeDBSubnetGroupsOutput, m.DescribeDBSubnetGroupError
}

func (m mockedDBSubnetGroups) DeleteDBSubnetGroup(*rds.DeleteDBSubnetGroupInput) (*rds.DeleteDBSubnetGroupOutput, error) {
	return &m.DeleteDBSubnetGroupOutput, nil
}

func TestDBSubnetGroups_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName1 := "test-db-subnet-group1"
	testName2 := "test-db-subnet-group2"
	dsg := DBSubnetGroups{
		Client: mockedDBSubnetGroups{
			DescribeDBSubnetGroupsOutput: rds.DescribeDBSubnetGroupsOutput{
				DBSubnetGroups: []*rds.DBSubnetGroup{
					{
						DBSubnetGroupName: awsgo.String(testName1),
					},
					{
						DBSubnetGroupName: awsgo.String(testName2),
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
			names, err := dsg.getAll(config.Config{
				DBSubnetGroups: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}

}

func TestDBSubnetGroups_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	dsg := DBSubnetGroups{
		Client: mockedDBSubnetGroups{
			DeleteDBSubnetGroupOutput:  rds.DeleteDBSubnetGroupOutput{},
			DescribeDBSubnetGroupError: awserr.New(rds.ErrCodeDBSubnetGroupNotFoundFault, "", nil),
		},
	}

	err := dsg.nukeAll([]*string{awsgo.String("test")})
	require.NoError(t, err)
}
