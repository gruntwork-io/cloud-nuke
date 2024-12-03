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

type mockedRdsSnapshot struct {
	RdsSnapshotAPI
	DescribeDBSnapshotsPagesOutput rds.DescribeDBSnapshotsOutput
	DeleteDBSnapshotOutput         rds.DeleteDBSnapshotOutput
}

func (m mockedRdsSnapshot) DescribeDBSnapshots(ctx context.Context, params *rds.DescribeDBSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSnapshotsOutput, error) {
	return &m.DescribeDBSnapshotsPagesOutput, nil
}

func (m mockedRdsSnapshot) DeleteDBSnapshot(ctx context.Context, params *rds.DeleteDBSnapshotInput, optFns ...func(*rds.Options)) (*rds.DeleteDBSnapshotOutput, error) {
	return &m.DeleteDBSnapshotOutput, nil
}

func TestRdsSnapshot_GetAll(t *testing.T) {

	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	now := time.Now()
	snapshot := RdsSnapshot{
		Client: mockedRdsSnapshot{
			DescribeDBSnapshotsPagesOutput: rds.DescribeDBSnapshotsOutput{
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
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := snapshot.getAll(context.Background(), config.Config{
				RdsSnapshot: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestRdsSnapshot_NukeAll(t *testing.T) {

	t.Parallel()

	testName := "test-db-cluster"
	snapshot := RdsSnapshot{
		Client: mockedRdsSnapshot{
			DeleteDBSnapshotOutput: rds.DeleteDBSnapshotOutput{},
		},
	}

	err := snapshot.nukeAll([]*string{&testName})
	assert.NoError(t, err)
}
