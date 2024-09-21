package resources

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedEventBridgeScheduleService struct {
	EventBridgeScheduleAPI

	ListSchedulesOutput  scheduler.ListSchedulesOutput
	DeleteScheduleOutput scheduler.DeleteScheduleOutput
}

func (m mockedEventBridgeScheduleService) DeleteSchedule(ctx context.Context, params *scheduler.DeleteScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.DeleteScheduleOutput, error) {
	return &m.DeleteScheduleOutput, nil
}

func (m mockedEventBridgeScheduleService) ListSchedules(ctx context.Context, params *scheduler.ListSchedulesInput, optFns ...func(*scheduler.Options)) (*scheduler.ListSchedulesOutput, error) {
	return &m.ListSchedulesOutput, nil
}

func Test_EventBridgeSchedule_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()

	schedule1 := "test-group-1"
	schedule2 := "test-group-2"

	group := "test-default"
	gSchedule1 := fmt.Sprintf("%s|%s", group, schedule1)
	gSchedule2 := fmt.Sprintf("%s|%s", group, schedule2)

	service := EventBridgeSchedule{
		Client: mockedEventBridgeScheduleService{
			ListSchedulesOutput: scheduler.ListSchedulesOutput{
				Schedules: []types.ScheduleSummary{
					{
						GroupName:    aws.String(group),
						Name:         aws.String(schedule1),
						CreationDate: &now,
					},
					{
						GroupName:    aws.String(group),
						Name:         aws.String(schedule2),
						CreationDate: aws.Time(now.Add(time.Hour)),
					},
				},
			}}}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{gSchedule1, gSchedule2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(schedule1),
					}},
				}},
			expected: []string{gSchedule2},
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
			buses, err := service.getAll(
				context.Background(),
				config.Config{EventBridgeSchedule: tc.configObj},
			)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(buses))
		})
	}
}

func Test_EventBridgeSchedule_NukeAll(t *testing.T) {
	t.Parallel()

	scheduleName := "test-schedule"
	service := EventBridgeSchedule{Client: mockedEventBridgeScheduleService{
		DeleteScheduleOutput: scheduler.DeleteScheduleOutput{},
	}}

	err := service.nukeAll([]*string{&scheduleName})
	assert.NoError(t, err)
}
