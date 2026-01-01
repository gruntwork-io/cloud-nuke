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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockCloudWatchAlarmsClient struct {
	DescribeAlarmsOutput    cloudwatch.DescribeAlarmsOutput
	DeleteAlarmsOutput      cloudwatch.DeleteAlarmsOutput
	PutCompositeAlarmOutput cloudwatch.PutCompositeAlarmOutput
}

func (m *mockCloudWatchAlarmsClient) DescribeAlarms(ctx context.Context, params *cloudwatch.DescribeAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsOutput, error) {
	return &m.DescribeAlarmsOutput, nil
}

func (m *mockCloudWatchAlarmsClient) DeleteAlarms(ctx context.Context, params *cloudwatch.DeleteAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DeleteAlarmsOutput, error) {
	return &m.DeleteAlarmsOutput, nil
}

func (m *mockCloudWatchAlarmsClient) PutCompositeAlarm(ctx context.Context, params *cloudwatch.PutCompositeAlarmInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.PutCompositeAlarmOutput, error) {
	return &m.PutCompositeAlarmOutput, nil
}

func TestListCloudWatchAlarms(t *testing.T) {
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	now := time.Now()

	mock := &mockCloudWatchAlarmsClient{
		DescribeAlarmsOutput: cloudwatch.DescribeAlarmsOutput{
			MetricAlarms: []types.MetricAlarm{
				{AlarmName: aws.String(testName1), AlarmConfigurationUpdatedTimestamp: &now},
				{AlarmName: aws.String(testName2), AlarmConfigurationUpdatedTimestamp: aws.Time(now.Add(1))},
			},
		},
	}

	names, err := listCloudWatchAlarms(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{testName1, testName2}, aws.ToStringSlice(names))
}

func TestListCloudWatchAlarms_WithFilter(t *testing.T) {
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	now := time.Now()

	mock := &mockCloudWatchAlarmsClient{
		DescribeAlarmsOutput: cloudwatch.DescribeAlarmsOutput{
			MetricAlarms: []types.MetricAlarm{
				{AlarmName: aws.String(testName1), AlarmConfigurationUpdatedTimestamp: &now},
				{AlarmName: aws.String(testName2), AlarmConfigurationUpdatedTimestamp: &now},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
		},
	}

	names, err := listCloudWatchAlarms(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{testName2}, aws.ToStringSlice(names))
}

func TestListCloudWatchAlarms_TimeFilter(t *testing.T) {
	t.Parallel()

	testName1 := "test-name1"
	now := time.Now()

	mock := &mockCloudWatchAlarmsClient{
		DescribeAlarmsOutput: cloudwatch.DescribeAlarmsOutput{
			MetricAlarms: []types.MetricAlarm{
				{AlarmName: aws.String(testName1), AlarmConfigurationUpdatedTimestamp: &now},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			TimeAfter: aws.Time(now.Add(-1)),
		},
	}

	names, err := listCloudWatchAlarms(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Empty(t, names)
}

func TestDeleteCompositeAlarm(t *testing.T) {
	t.Parallel()

	mock := &mockCloudWatchAlarmsClient{}
	err := deleteCompositeAlarm(context.Background(), mock, aws.String("test-alarm"))
	require.NoError(t, err)
}

func TestDeleteMetricAlarmsBulk(t *testing.T) {
	t.Parallel()

	mock := &mockCloudWatchAlarmsClient{}
	err := deleteMetricAlarmsBulk(context.Background(), mock, []string{"alarm1", "alarm2"})
	require.NoError(t, err)
}

func TestNukeCloudWatchAlarms_MetricOnly(t *testing.T) {
	t.Parallel()

	testName1 := "test-name1"
	testName2 := "test-name2"
	now := time.Now()

	mock := &mockCloudWatchAlarmsClient{
		DescribeAlarmsOutput: cloudwatch.DescribeAlarmsOutput{
			MetricAlarms: []types.MetricAlarm{
				{AlarmName: aws.String(testName1), AlarmConfigurationUpdatedTimestamp: &now},
				{AlarmName: aws.String(testName2), AlarmConfigurationUpdatedTimestamp: aws.Time(now.Add(1))},
			},
		},
	}

	err := nukeCloudWatchAlarms(context.Background(), mock, resource.Scope{}, "cloudwatch-alarm", []*string{aws.String(testName1), aws.String(testName2)})
	require.NoError(t, err)
}

func TestNukeCloudWatchAlarms_CompositeAndMetric(t *testing.T) {
	t.Parallel()

	testCompositeAlarm1 := "test-composite-1"
	testCompositeAlarm2 := "test-composite-2"
	testMetricAlarm := "test-metric"
	now := time.Now()

	mock := &mockCloudWatchAlarmsClient{
		DescribeAlarmsOutput: cloudwatch.DescribeAlarmsOutput{
			MetricAlarms: []types.MetricAlarm{
				{AlarmName: aws.String(testMetricAlarm), AlarmConfigurationUpdatedTimestamp: &now},
			},
			CompositeAlarms: []types.CompositeAlarm{
				{AlarmName: aws.String(testCompositeAlarm1), AlarmConfigurationUpdatedTimestamp: &now},
				{AlarmName: aws.String(testCompositeAlarm2), AlarmConfigurationUpdatedTimestamp: &now},
			},
		},
	}

	err := nukeCloudWatchAlarms(context.Background(), mock, resource.Scope{}, "cloudwatch-alarm", []*string{
		aws.String(testCompositeAlarm1),
		aws.String(testCompositeAlarm2),
		aws.String(testMetricAlarm),
	})
	require.NoError(t, err)
}
