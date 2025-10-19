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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedDBInstance struct {
	t *testing.T

	RDSAPI
	DescribeDBInstancesOutput rds.DescribeDBInstancesOutput
	DeleteDBInstanceOutput    rds.DeleteDBInstanceOutput
	ModifyCallExpected        bool
	InstancesDeleted          map[string]bool
}

func (m mockedDBInstance) ModifyDBInstance(ctx context.Context, params *rds.ModifyDBInstanceInput, optFns ...func(*rds.Options)) (*rds.ModifyDBInstanceOutput, error) {
	if !m.ModifyCallExpected {
		assert.Fail(m.t, "ModifyDBInstance should not be called for cluster member instances")
	}
	assert.NotNil(m.t, params)
	assert.NotEmpty(m.t, *params.DBInstanceIdentifier)
	assert.False(m.t, *params.DeletionProtection)
	assert.True(m.t, *params.ApplyImmediately)
	return nil, nil
}

func (m mockedDBInstance) DescribeDBInstances(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
	// If specific instance is requested and it's been deleted, return empty result
	if params.DBInstanceIdentifier != nil {
		if m.InstancesDeleted != nil && m.InstancesDeleted[*params.DBInstanceIdentifier] {
			return &rds.DescribeDBInstancesOutput{DBInstances: []types.DBInstance{}}, nil
		}
	}
	return &m.DescribeDBInstancesOutput, nil
}

func (m mockedDBInstance) DeleteDBInstance(ctx context.Context, params *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error) {
	// Mark instance as deleted for waiter
	if m.InstancesDeleted != nil && params.DBInstanceIdentifier != nil {
		m.InstancesDeleted[*params.DBInstanceIdentifier] = true
	}
	return &m.DeleteDBInstanceOutput, nil
}

func (m mockedDBInstance) WaitForOutput(ctx context.Context, params *rds.DescribeDBInstancesInput, maxWaitDur time.Duration, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
	return nil, nil
}

func TestDBInstances_GetAll(t *testing.T) {

	t.Parallel()

	testName1 := "test-db-instance1"
	testName2 := "test-db-instance2"
	testName3 := "test-db-instance3"
	testIdentifier1 := "test-identifier1"
	testIdentifier2 := "test-identifier2"
	testIdentifier3 := "test-identifier3"
	now := time.Now()
	di := DBInstances{
		Client: mockedDBInstance{
			t: t,
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
					{
						DBInstanceIdentifier: aws.String(testIdentifier3),
						DBName:               aws.String(testName3),
						InstanceCreateTime:   aws.Time(now.Add(1)),
						DeletionProtection:   aws.Bool(true),
					},
				},
			},
		},
	}

	tests := map[string]struct {
		configObj config.AWSProtectectableResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.AWSProtectectableResourceType{},
			expected:  []string{testIdentifier1, testIdentifier2},
		},
		"nameExclusionFilter": {
			configObj: config.AWSProtectectableResourceType{ResourceType: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile("^" + testName1 + "$"),
					}},
				},
			}},
			expected: []string{testIdentifier1, testIdentifier2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.AWSProtectectableResourceType{
				ResourceType: config.ResourceType{
					ExcludeRule: config.FilterRule{
						TimeAfter: aws.Time(now),
					}}},
			expected: []string{testIdentifier1},
		},
		"includeDeletionProtection": {
			configObj: config.AWSProtectectableResourceType{
				IncludeDeletionProtected: true,
				ResourceType:             config.ResourceType{},
			},
			expected: []string{testIdentifier1, testIdentifier2, testIdentifier3},
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

	t.Run("standalone instance", func(t *testing.T) {
		di := DBInstances{
			Client: mockedDBInstance{
				t: t,
				DescribeDBInstancesOutput: rds.DescribeDBInstancesOutput{
					DBInstances: []types.DBInstance{
						{
							DBInstanceIdentifier: aws.String("test-standalone"),
							DBClusterIdentifier:  nil, // Not part of a cluster
						},
					},
				},
				DeleteDBInstanceOutput: rds.DeleteDBInstanceOutput{},
				ModifyCallExpected:     true,                  // Should call ModifyDBInstance
				InstancesDeleted:       make(map[string]bool), // Track deleted instances
			},
		}
		di.Context = context.Background()

		err := di.nukeAll([]*string{aws.String("test-standalone")})
		require.NoError(t, err)
	})

	t.Run("cluster member instance", func(t *testing.T) {
		di := DBInstances{
			Client: mockedDBInstance{
				t: t,
				DescribeDBInstancesOutput: rds.DescribeDBInstancesOutput{
					DBInstances: []types.DBInstance{
						{
							DBInstanceIdentifier: aws.String("test-cluster-member"),
							DBClusterIdentifier:  aws.String("my-aurora-cluster"), // Part of a cluster
						},
					},
				},
				DeleteDBInstanceOutput: rds.DeleteDBInstanceOutput{},
				ModifyCallExpected:     false,                 // Should NOT call ModifyDBInstance
				InstancesDeleted:       make(map[string]bool), // Track deleted instances
			},
		}
		di.Context = context.Background()

		err := di.nukeAll([]*string{aws.String("test-cluster-member")})
		require.NoError(t, err)
	})
}
