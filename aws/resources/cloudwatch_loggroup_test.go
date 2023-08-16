package resources

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

type mockedCloudWatchLogGroup struct {
	cloudwatchlogsiface.CloudWatchLogsAPI
	DeleteLogGroupOutput    cloudwatchlogs.DeleteLogGroupOutput
	DescribeLogGroupsOutput cloudwatchlogs.DescribeLogGroupsOutput
}

func (m mockedCloudWatchLogGroup) DescribeLogGroupsPages(input *cloudwatchlogs.DescribeLogGroupsInput, fn func(*cloudwatchlogs.DescribeLogGroupsOutput, bool) bool) error {
	fn(&m.DescribeLogGroupsOutput, true)
	return nil
}

func (m mockedCloudWatchLogGroup) DeleteLogGroup(input *cloudwatchlogs.DeleteLogGroupInput) (*cloudwatchlogs.DeleteLogGroupOutput, error) {
	return &m.DeleteLogGroupOutput, nil
}

func TestCloudWatchLogGroup_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	now := time.Now()
	cw := CloudWatchLogGroups{
		Client: mockedCloudWatchLogGroup{
			DescribeLogGroupsOutput: cloudwatchlogs.DescribeLogGroupsOutput{
				LogGroups: []*cloudwatchlogs.LogGroup{
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
			names, err := cw.getAll(config.Config{
				CloudWatchLogGroup: tc.configObj,
			})

			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestCloudWatchLogGroup_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()
	cw := CloudWatchLogGroups{
		Client: mockedCloudWatchLogGroup{
			DeleteLogGroupOutput: cloudwatchlogs.DeleteLogGroupOutput{},
		}}

	err := cw.nukeAll([]*string{aws.String("test-name1"), aws.String("test-name2")})
	require.NoError(t, err)
}
