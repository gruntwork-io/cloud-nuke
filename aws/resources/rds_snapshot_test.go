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
					DBSnapshotIdentifier: aws.String(testName1),
					SnapshotCreateTime:   aws.Time(now),
				},
				{
					DBSnapshotIdentifier: aws.String(testName2),
					SnapshotCreateTime:   aws.Time(now.Add(1 * time.Hour)),
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
			names, err := listRdsSnapshots(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteRdsSnapshot(t *testing.T) {
	t.Parallel()

	mock := &mockRdsSnapshotClient{}
	err := deleteRdsSnapshot(context.Background(), mock, aws.String("test-snapshot"))
	require.NoError(t, err)
}
