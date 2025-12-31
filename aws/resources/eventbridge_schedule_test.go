package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEventBridgeScheduleClient struct {
	ListSchedulesOutput  scheduler.ListSchedulesOutput
	DeleteScheduleOutput scheduler.DeleteScheduleOutput
}

func (m *mockEventBridgeScheduleClient) ListSchedules(ctx context.Context, params *scheduler.ListSchedulesInput, optFns ...func(*scheduler.Options)) (*scheduler.ListSchedulesOutput, error) {
	return &m.ListSchedulesOutput, nil
}

func (m *mockEventBridgeScheduleClient) DeleteSchedule(ctx context.Context, params *scheduler.DeleteScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.DeleteScheduleOutput, error) {
	return &m.DeleteScheduleOutput, nil
}

func TestEventBridgeSchedule_ResourceName(t *testing.T) {
	r := NewEventBridgeSchedule()
	assert.Equal(t, "event-bridge-schedule", r.ResourceName())
}

func TestEventBridgeSchedule_MaxBatchSize(t *testing.T) {
	r := NewEventBridgeSchedule()
	assert.Equal(t, 100, r.MaxBatchSize())
}

func TestListEventBridgeSchedules(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockEventBridgeScheduleClient{
		ListSchedulesOutput: scheduler.ListSchedulesOutput{
			Schedules: []types.ScheduleSummary{
				{Name: aws.String("schedule1"), GroupName: aws.String("default"), CreationDate: aws.Time(now)},
				{Name: aws.String("schedule2"), GroupName: aws.String("custom"), CreationDate: aws.Time(now)},
			},
		},
	}

	names, err := listEventBridgeSchedules(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"default|schedule1", "custom|schedule2"}, aws.ToStringSlice(names))
}

func TestListEventBridgeSchedules_WithFilter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockEventBridgeScheduleClient{
		ListSchedulesOutput: scheduler.ListSchedulesOutput{
			Schedules: []types.ScheduleSummary{
				{Name: aws.String("schedule1"), GroupName: aws.String("default"), CreationDate: aws.Time(now)},
				{Name: aws.String("skip-this"), GroupName: aws.String("default"), CreationDate: aws.Time(now)},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(".*skip-.*")}},
		},
	}

	names, err := listEventBridgeSchedules(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"default|schedule1"}, aws.ToStringSlice(names))
}

func TestDeleteEventBridgeSchedule(t *testing.T) {
	t.Parallel()

	mock := &mockEventBridgeScheduleClient{}
	err := deleteEventBridgeSchedule(context.Background(), mock, aws.String("default|test-schedule"))
	require.NoError(t, err)
}
