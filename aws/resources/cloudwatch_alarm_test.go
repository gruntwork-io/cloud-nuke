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

func (m *mockCloudWatchAlarmsClient) ListTagsForResource(ctx context.Context, params *cloudwatch.ListTagsForResourceInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.ListTagsForResourceOutput, error) {
	if aws.ToString(params.ResourceARN) == "arn:aws:cloudwatch:us-east-1:123456789:alarm:test-alarm-2" {
		return &cloudwatch.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("env"), Value: aws.String("prod")}}}, nil
	}
	return &cloudwatch.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("env"), Value: aws.String("dev")}}}, nil
}

func TestListCloudWatchAlarms(t *testing.T) {
	t.Parallel()

	testName1 := "test-alarm-1"
	testName2 := "test-alarm-2"
	now := time.Now()

	mock := &mockCloudWatchAlarmsClient{
		DescribeAlarmsOutput: cloudwatch.DescribeAlarmsOutput{
			MetricAlarms: []types.MetricAlarm{
				{AlarmName: aws.String(testName1), AlarmArn: aws.String("arn:aws:cloudwatch:us-east-1:123456789:alarm:test-alarm-1"), AlarmConfigurationUpdatedTimestamp: &now},
				{AlarmName: aws.String(testName2), AlarmArn: aws.String("arn:aws:cloudwatch:us-east-1:123456789:alarm:test-alarm-2"), AlarmConfigurationUpdatedTimestamp: aws.Time(now.Add(1 * time.Hour))},
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
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{testName1},
		},
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"env": {RE: *regexp.MustCompile("^prod$")},
					},
				},
			},
			expected: []string{testName2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listCloudWatchAlarms(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteCloudWatchAlarm_MetricAlarm(t *testing.T) {
	t.Parallel()

	mock := &mockCloudWatchAlarmsClient{
		DescribeAlarmsOutput: cloudwatch.DescribeAlarmsOutput{
			CompositeAlarms: []types.CompositeAlarm{}, // Not a composite alarm
		},
	}

	err := deleteCloudWatchAlarm(context.Background(), mock, aws.String("metric-alarm"))
	require.NoError(t, err)
}

func TestDeleteCloudWatchAlarm_CompositeAlarm(t *testing.T) {
	t.Parallel()

	mock := &mockCloudWatchAlarmsClient{
		DescribeAlarmsOutput: cloudwatch.DescribeAlarmsOutput{
			CompositeAlarms: []types.CompositeAlarm{
				{AlarmName: aws.String("composite-alarm")},
			},
		},
	}

	err := deleteCloudWatchAlarm(context.Background(), mock, aws.String("composite-alarm"))
	require.NoError(t, err)
}
