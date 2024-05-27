package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/gruntwork-io/cloud-nuke/config"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"

	"github.com/stretchr/testify/require"
)

type mockedDBInstance struct {
	rdsiface.RDSAPI
	DescribeDBInstancesOutput rds.DescribeDBInstancesOutput
	DeleteDBInstanceOutput    rds.DeleteDBInstanceOutput
}

func (m mockedDBInstance) DescribeDBInstancesWithContext(_ awsgo.Context, _ *rds.DescribeDBInstancesInput, _ ...request.Option) (*rds.DescribeDBInstancesOutput, error) {
	return &m.DescribeDBInstancesOutput, nil
}

func (m mockedDBInstance) DeleteDBInstanceWithContext(_ awsgo.Context, _ *rds.DeleteDBInstanceInput, _ ...request.Option) (*rds.DeleteDBInstanceOutput, error) {
	return &m.DeleteDBInstanceOutput, nil
}

func (m mockedDBInstance) WaitUntilDBInstanceDeletedWithContext(_ awsgo.Context, _ *rds.DescribeDBInstancesInput, _ ...request.WaiterOption) error {
	return nil
}

func TestDBInstances_GetAll(t *testing.T) {

	t.Parallel()

	testName1 := "test-db-instance1"
	testName2 := "test-db-instance2"
	testIdentifier1 := "test-identifier1"
	testIdentifier2 := "test-identifier2"
	now := time.Now()
	di := DBInstances{
		Client: mockedDBInstance{
			DescribeDBInstancesOutput: rds.DescribeDBInstancesOutput{
				DBInstances: []*rds.DBInstance{
					{
						DBInstanceIdentifier: awsgo.String(testIdentifier1),
						DBName:               awsgo.String(testName1),
						InstanceCreateTime:   awsgo.Time(now),
					},
					{
						DBInstanceIdentifier: awsgo.String(testIdentifier2),
						DBName:               awsgo.String(testName2),
						InstanceCreateTime:   awsgo.Time(now.Add(1)),
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
			expected:  []string{testIdentifier1, testIdentifier2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testIdentifier2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testIdentifier1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := di.getAll(context.Background(), config.Config{
				DBInstances: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}

}

func TestDBInstances_NukeAll(t *testing.T) {

	t.Parallel()

	di := DBInstances{
		Client: mockedDBInstance{
			DeleteDBInstanceOutput: rds.DeleteDBInstanceOutput{},
		},
	}

	err := di.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
