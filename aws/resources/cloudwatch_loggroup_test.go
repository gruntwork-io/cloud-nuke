package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedCloudWatchLogGroup struct {
	CloudWatchLogGroupsAPI
	DescribeLogGroupsOutput cloudwatchlogs.DescribeLogGroupsOutput
	DeleteLogGroupOutput    cloudwatchlogs.DeleteLogGroupOutput
}

func (m mockedCloudWatchLogGroup) DescribeLogGroups(ctx context.Context, params *cloudwatchlogs.DescribeLogGroupsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	return &m.DescribeLogGroupsOutput, nil
}

func (m mockedCloudWatchLogGroup) DeleteLogGroup(ctx context.Context, params *cloudwatchlogs.DeleteLogGroupInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DeleteLogGroupOutput, error) {
	return &m.DeleteLogGroupOutput, nil
}

func TestCloudWatchLogGroup_GetAll(t *testing.T) {
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	now := time.Now()
	cw := CloudWatchLogGroups{
		Client: mockedCloudWatchLogGroup{
			DescribeLogGroupsOutput: cloudwatchlogs.DescribeLogGroupsOutput{
				LogGroups: []types.LogGroup{
					{
						LogGroupName: aws.String(testName1),
						CreationTime: aws.Int64(now.UnixMilli()),
					},
					{
						LogGroupName: aws.String(testName2),
						CreationTime: aws.Int64(now.Add(1).UnixMilli()),
					},
				},
			},
		}}

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
					TimeAfter: aws.Time(now.Add(-2 * time.Hour)),
				}},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := cw.getAll(context.Background(), config.Config{
				CloudWatchLogGroup: tc.configObj,
			})

			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestCloudWatchLogGroup_NukeAll(t *testing.T) {
	t.Parallel()
	cw := CloudWatchLogGroups{
		Client: mockedCloudWatchLogGroup{
			DeleteLogGroupOutput: cloudwatchlogs.DeleteLogGroupOutput{},
		}}

	err := cw.nukeAll([]*string{aws.String("test-name1"), aws.String("test-name2")})
	require.NoError(t, err)
}
