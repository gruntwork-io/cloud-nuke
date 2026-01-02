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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockedDBInstances struct {
	DBInstancesAPI
	DescribeDBInstancesOutput rds.DescribeDBInstancesOutput
	ModifyDBInstanceOutput    rds.ModifyDBInstanceOutput
	DeleteDBInstanceOutput    rds.DeleteDBInstanceOutput
	ModifyCallCount           int
	DeleteCallCount           int
}

func (m *mockedDBInstances) DescribeDBInstances(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
	return &m.DescribeDBInstancesOutput, nil
}

func (m *mockedDBInstances) ModifyDBInstance(ctx context.Context, params *rds.ModifyDBInstanceInput, optFns ...func(*rds.Options)) (*rds.ModifyDBInstanceOutput, error) {
	m.ModifyCallCount++
	return &m.ModifyDBInstanceOutput, nil
}

func (m *mockedDBInstances) DeleteDBInstance(ctx context.Context, params *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error) {
	m.DeleteCallCount++
	return &m.DeleteDBInstanceOutput, nil
}

func TestDBInstances_GetAll(t *testing.T) {
	t.Parallel()

	testID1 := "db-instance-1"
	testID2 := "db-instance-2"
	now := time.Now()

	mock := &mockedDBInstances{
		DescribeDBInstancesOutput: rds.DescribeDBInstancesOutput{
			DBInstances: []types.DBInstance{
				{
					DBInstanceIdentifier: aws.String(testID1),
					InstanceCreateTime:   aws.Time(now),
					TagList:              []types.Tag{{Key: aws.String("env"), Value: aws.String("dev")}},
				},
				{
					DBInstanceIdentifier: aws.String(testID2),
					InstanceCreateTime:   aws.Time(now.Add(1 * time.Hour)),
					TagList:              []types.Tag{{Key: aws.String("env"), Value: aws.String("prod")}},
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
			expected:  []string{testID1, testID2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile("db-instance-1"),
					}},
				},
			},
			expected: []string{testID2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{testID1},
		},
		"tagExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"env": {RE: *regexp.MustCompile("prod")},
					},
				},
			},
			expected: []string{testID1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listDBInstances(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDBInstances_Delete_Standalone(t *testing.T) {
	t.Parallel()

	// Standalone instance (not part of cluster) - should call ModifyDBInstance
	mock := &mockedDBInstances{
		DescribeDBInstancesOutput: rds.DescribeDBInstancesOutput{
			DBInstances: []types.DBInstance{
				{
					DBInstanceIdentifier: aws.String("standalone-db"),
					DBClusterIdentifier:  nil, // Not part of a cluster
				},
			},
		},
	}

	err := deleteDBInstance(context.Background(), mock, aws.String("standalone-db"))
	require.NoError(t, err)
	require.Equal(t, 1, mock.ModifyCallCount, "ModifyDBInstance should be called for standalone instances")
	require.Equal(t, 1, mock.DeleteCallCount, "DeleteDBInstance should be called")
}

func TestDBInstances_Delete_ClusterMember(t *testing.T) {
	t.Parallel()

	// Cluster member instance - should NOT call ModifyDBInstance
	mock := &mockedDBInstances{
		DescribeDBInstancesOutput: rds.DescribeDBInstancesOutput{
			DBInstances: []types.DBInstance{
				{
					DBInstanceIdentifier: aws.String("cluster-member-db"),
					DBClusterIdentifier:  aws.String("my-aurora-cluster"), // Part of a cluster
				},
			},
		},
	}

	err := deleteDBInstance(context.Background(), mock, aws.String("cluster-member-db"))
	require.NoError(t, err)
	require.Equal(t, 0, mock.ModifyCallCount, "ModifyDBInstance should NOT be called for cluster members")
	require.Equal(t, 1, mock.DeleteCallCount, "DeleteDBInstance should be called")
}
