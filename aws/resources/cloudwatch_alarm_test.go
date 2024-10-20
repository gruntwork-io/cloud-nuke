package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedCloudWatchAlarms struct {
	CloudWatchAlarmsAPI
	DescribeAlarmsOutput    cloudwatch.DescribeAlarmsOutput
	DeleteAlarmsOutput      cloudwatch.DeleteAlarmsOutput
	PutCompositeAlarmOutput cloudwatch.PutCompositeAlarmOutput
}

func (m mockedCloudWatchAlarms) DescribeAlarms(ctx context.Context, params *cloudwatch.DescribeAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsOutput, error) {
	return &m.DescribeAlarmsOutput, nil
}

func (m mockedCloudWatchAlarms) DeleteAlarms(ctx context.Context, params *cloudwatch.DeleteAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DeleteAlarmsOutput, error) {
	return &m.DeleteAlarmsOutput, nil
}

func (m mockedCloudWatchAlarms) PutCompositeAlarm(ctx context.Context, params *cloudwatch.PutCompositeAlarmInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.PutCompositeAlarmOutput, error) {
	return &m.PutCompositeAlarmOutput, nil
}

func TestCloudWatchAlarm_GetAll(t *testing.T) {

	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	now := time.Now()
	cw := CloudWatchAlarms{
		Client: mockedCloudWatchAlarms{
			DescribeAlarmsOutput: cloudwatch.DescribeAlarmsOutput{
				MetricAlarms: []types.MetricAlarm{
					{AlarmName: aws.String(testName1), AlarmConfigurationUpdatedTimestamp: &now},
					{AlarmName: aws.String(testName2), AlarmConfigurationUpdatedTimestamp: aws.Time(now.Add(1))},
				}},
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
					TimeAfter: aws.Time(now.Add(-1)),
				}},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := cw.getAll(context.Background(), config.Config{
				CloudWatchAlarm: tc.configObj,
			})

			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestCloudWatchAlarms_NukeAll(t *testing.T) {
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	now := time.Now()
	cw := CloudWatchAlarms{
		Client: mockedCloudWatchAlarms{
			DescribeAlarmsOutput: cloudwatch.DescribeAlarmsOutput{
				MetricAlarms: []types.MetricAlarm{
					{AlarmName: aws.String(testName1), AlarmConfigurationUpdatedTimestamp: &now},
					{AlarmName: aws.String(testName2), AlarmConfigurationUpdatedTimestamp: aws.Time(now.Add(1))},
				}},
			PutCompositeAlarmOutput: cloudwatch.PutCompositeAlarmOutput{},
			DeleteAlarmsOutput:      cloudwatch.DeleteAlarmsOutput{},
		}}

	err := cw.nukeAll([]*string{aws.String(testName1), aws.String(testName2)})
	require.NoError(t, err)
}

func TestCloudWatchCompositeAlarms_NukeAll(t *testing.T) {
	t.Parallel()

	testCompositeAlaram1 := "test-name1"
	testCompositeAlaram2 := "test-name2"
	testName3 := "test-name3"
	now := time.Now()
	cw := CloudWatchAlarms{
		Client: mockedCloudWatchAlarms{
			DescribeAlarmsOutput: cloudwatch.DescribeAlarmsOutput{
				MetricAlarms: []types.MetricAlarm{
					{AlarmName: aws.String(testCompositeAlaram1), AlarmConfigurationUpdatedTimestamp: &now},
					{AlarmName: aws.String(testCompositeAlaram2), AlarmConfigurationUpdatedTimestamp: &now},
				}},
			PutCompositeAlarmOutput: cloudwatch.PutCompositeAlarmOutput{},
			DeleteAlarmsOutput:      cloudwatch.DeleteAlarmsOutput{},
		}}

	err := cw.nukeAll([]*string{aws.String(testCompositeAlaram1), aws.String(testCompositeAlaram2), aws.String(testName3)})
	require.NoError(t, err)
}
