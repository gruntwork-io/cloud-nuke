package aws

import (
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedCloudWatchAlarms struct {
	cloudwatchiface.CloudWatchAPI
	DescribeAlarmsOutput    cloudwatch.DescribeAlarmsOutput
	DeleteAlarmsOutput      cloudwatch.DeleteAlarmsOutput
	PutCompositeAlarmOutput cloudwatch.PutCompositeAlarmOutput
}

func (m mockedCloudWatchAlarms) DescribeAlarmsPages(input *cloudwatch.DescribeAlarmsInput, fn func(*cloudwatch.DescribeAlarmsOutput, bool) bool) error {
	fn(&m.DescribeAlarmsOutput, true)
	return nil
}

func (m mockedCloudWatchAlarms) PutCompositeAlarm(input *cloudwatch.PutCompositeAlarmInput) (*cloudwatch.PutCompositeAlarmOutput, error) {
	return &m.PutCompositeAlarmOutput, nil
}

func (m mockedCloudWatchAlarms) DescribeAlarms(input *cloudwatch.DescribeAlarmsInput) (*cloudwatch.DescribeAlarmsOutput, error) {
	return &m.DescribeAlarmsOutput, nil
}

func (m mockedCloudWatchAlarms) DeleteAlarms(input *cloudwatch.DeleteAlarmsInput) (*cloudwatch.DeleteAlarmsOutput, error) {
	return &m.DeleteAlarmsOutput, nil
}

func TestCloudWatchAlarm_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	now := time.Now()
	cw := CloudWatchAlarms{
		Client: mockedCloudWatchAlarms{
			DescribeAlarmsOutput: cloudwatch.DescribeAlarmsOutput{
				MetricAlarms: []*cloudwatch.MetricAlarm{
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
			names, err := cw.getAll(config.Config{
				CloudWatchAlarm: tc.configObj,
			})

			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestCloudWatchAlarms_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	now := time.Now()
	cw := CloudWatchAlarms{
		Client: mockedCloudWatchAlarms{
			DescribeAlarmsOutput: cloudwatch.DescribeAlarmsOutput{
				MetricAlarms: []*cloudwatch.MetricAlarm{
					{AlarmName: aws.String(testName1), AlarmConfigurationUpdatedTimestamp: &now},
					{AlarmName: aws.String(testName2), AlarmConfigurationUpdatedTimestamp: aws.Time(now.Add(1))},
				}},
			PutCompositeAlarmOutput: cloudwatch.PutCompositeAlarmOutput{},
			DeleteAlarmsOutput:      cloudwatch.DeleteAlarmsOutput{},
		}}

	err := cw.nukeAll([]*string{aws.String(testName1), aws.String(testName2)})
	require.NoError(t, err)
}
