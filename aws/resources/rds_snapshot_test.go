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

type mockRdsSnapshotClient struct {
	DescribeDBSnapshotsOutput rds.DescribeDBSnapshotsOutput
	DeleteDBSnapshotOutput    rds.DeleteDBSnapshotOutput
}

func (m *mockRdsSnapshotClient) DescribeDBSnapshots(ctx context.Context, params *rds.DescribeDBSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSnapshotsOutput, error) {
	return &m.DescribeDBSnapshotsOutput, nil
}

func (m *mockRdsSnapshotClient) DeleteDBSnapshot(ctx context.Context, params *rds.DeleteDBSnapshotInput, optFns ...func(*rds.Options)) (*rds.DeleteDBSnapshotOutput, error) {
	return &m.DeleteDBSnapshotOutput, nil
}

func TestListRdsSnapshots(t *testing.T) {
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	now := time.Now()

	mock := &mockRdsSnapshotClient{
		DescribeDBSnapshotsOutput: rds.DescribeDBSnapshotsOutput{
			DBSnapshots: []types.DBSnapshot{
				{
					DBSnapshotIdentifier: &testName1,
					SnapshotCreateTime:   &now,
				},
				{
					DBSnapshotIdentifier: &testName2,
					SnapshotCreateTime:   aws.Time(now.Add(1)),
				},
			},
		},
	}

	names, err := listRdsSnapshots(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{testName1, testName2}, aws.ToStringSlice(names))
}

func TestListRdsSnapshots_WithNameExclusionFilter(t *testing.T) {
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	now := time.Now()

	mock := &mockRdsSnapshotClient{
		DescribeDBSnapshotsOutput: rds.DescribeDBSnapshotsOutput{
			DBSnapshots: []types.DBSnapshot{
				{
					DBSnapshotIdentifier: &testName1,
					SnapshotCreateTime:   &now,
				},
				{
					DBSnapshotIdentifier: &testName2,
					SnapshotCreateTime:   aws.Time(now.Add(1)),
				},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
		},
	}

	names, err := listRdsSnapshots(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{testName2}, aws.ToStringSlice(names))
}

func TestListRdsSnapshots_TimeAfterExclusionFilter(t *testing.T) {
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	now := time.Now()

	mock := &mockRdsSnapshotClient{
		DescribeDBSnapshotsOutput: rds.DescribeDBSnapshotsOutput{
			DBSnapshots: []types.DBSnapshot{
				{
					DBSnapshotIdentifier: &testName1,
					SnapshotCreateTime:   &now,
				},
				{
					DBSnapshotIdentifier: &testName2,
					SnapshotCreateTime:   aws.Time(now.Add(1)),
				},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
		},
	}

	names, err := listRdsSnapshots(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Empty(t, names)
}

func TestDeleteRdsSnapshot(t *testing.T) {
	t.Parallel()

	mock := &mockRdsSnapshotClient{}
	testName := "test-db-snapshot"
	err := deleteRdsSnapshot(context.Background(), mock, &testName)
	require.NoError(t, err)
}
