package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedDBInstance struct {
	RDSAPI
	DescribeDBInstancesOutput rds.DescribeDBInstancesOutput
	DeleteDBInstanceOutput    rds.DeleteDBInstanceOutput
}

func (m mockedDBInstance) DescribeDBInstances(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
	return &m.DescribeDBInstancesOutput, nil
}

func (m mockedDBInstance) DeleteDBInstance(ctx context.Context, params *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error) {
	return &m.DeleteDBInstanceOutput, nil
}
func (m mockedDBInstance) WaitForOutput(ctx context.Context, params *rds.DescribeDBInstancesInput, maxWaitDur time.Duration, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
	return nil, nil
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
				DBInstances: []types.DBInstance{
					{
						DBInstanceIdentifier: aws.String(testIdentifier1),
						DBName:               aws.String(testName1),
						InstanceCreateTime:   aws.Time(now),
					},
					{
						DBInstanceIdentifier: aws.String(testIdentifier2),
						DBName:               aws.String(testName2),
						InstanceCreateTime:   aws.Time(now.Add(1)),
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
						RE: *regexp.MustCompile("^" + testName1 + "$"),
					}},
				},
			},
			expected: []string{testIdentifier1, testIdentifier2},
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
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
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
	di.Context = context.Background()

	err := di.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
