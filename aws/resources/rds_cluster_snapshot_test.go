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

type mockRdsClusterSnapshotClient struct {
	DescribeDBClusterSnapshotsOutput rds.DescribeDBClusterSnapshotsOutput
	DeleteDBClusterSnapshotOutput    rds.DeleteDBClusterSnapshotOutput
}

func (m *mockRdsClusterSnapshotClient) DescribeDBClusterSnapshots(ctx context.Context, params *rds.DescribeDBClusterSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClusterSnapshotsOutput, error) {
	return &m.DescribeDBClusterSnapshotsOutput, nil
}

func (m *mockRdsClusterSnapshotClient) DeleteDBClusterSnapshot(ctx context.Context, params *rds.DeleteDBClusterSnapshotInput, optFns ...func(*rds.Options)) (*rds.DeleteDBClusterSnapshotOutput, error) {
	return &m.DeleteDBClusterSnapshotOutput, nil
}

func TestListRdsClusterSnapshots(t *testing.T) {
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	now := time.Now()

	mock := &mockRdsClusterSnapshotClient{
		DescribeDBClusterSnapshotsOutput: rds.DescribeDBClusterSnapshotsOutput{
			DBClusterSnapshots: []types.DBClusterSnapshot{
				{
					DBClusterSnapshotIdentifier: aws.String(testName1),
					SnapshotCreateTime:          aws.Time(now),
					SnapshotType:                aws.String("manual"),
				},
				{
					DBClusterSnapshotIdentifier: aws.String(testName2),
					SnapshotCreateTime:          aws.Time(now.Add(1 * time.Hour)),
					SnapshotType:                aws.String("manual"),
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
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				},
			},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listRdsClusterSnapshots(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestListRdsClusterSnapshots_SkipsAutomated(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockRdsClusterSnapshotClient{
		DescribeDBClusterSnapshotsOutput: rds.DescribeDBClusterSnapshotsOutput{
			DBClusterSnapshots: []types.DBClusterSnapshot{
				{
					DBClusterSnapshotIdentifier: aws.String("manual-snap"),
					SnapshotCreateTime:          aws.Time(now),
					SnapshotType:                aws.String("manual"),
				},
				{
					DBClusterSnapshotIdentifier: aws.String("auto-snap"),
					SnapshotCreateTime:          aws.Time(now),
					SnapshotType:                aws.String("automated"),
				},
				{
					DBClusterSnapshotIdentifier: aws.String("backup-snap"),
					SnapshotCreateTime:          aws.Time(now),
					SnapshotType:                aws.String("awsbackup"),
				},
			},
		},
	}

	names, err := listRdsClusterSnapshots(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.Equal(t, []string{"manual-snap", "backup-snap"}, aws.ToStringSlice(names))
}

func TestDeleteRdsClusterSnapshot(t *testing.T) {
	t.Parallel()

	mock := &mockRdsClusterSnapshotClient{}
	err := deleteRdsClusterSnapshot(context.Background(), mock, aws.String("test-snapshot"))
	require.NoError(t, err)
}
