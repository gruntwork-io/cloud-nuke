package resources

import (
	"context"
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

type mockedEventBridgeScheduleGroupService struct {
	EventBridgeScheduleGroupAPI
	ListScheduleGroupsOutput  scheduler.ListScheduleGroupsOutput
	DeleteScheduleGroupOutput scheduler.DeleteScheduleGroupOutput
}

func (m mockedEventBridgeScheduleGroupService) DeleteScheduleGroup(ctx context.Context, params *scheduler.DeleteScheduleGroupInput, optFns ...func(*scheduler.Options)) (*scheduler.DeleteScheduleGroupOutput, error) {
	return &m.DeleteScheduleGroupOutput, nil
}

func (m mockedEventBridgeScheduleGroupService) ListScheduleGroups(ctx context.Context, params *scheduler.ListScheduleGroupsInput, optFns ...func(*scheduler.Options)) (*scheduler.ListScheduleGroupsOutput, error) {
	return &m.ListScheduleGroupsOutput, nil
}

func Test_EventBridgeScheduleGroup_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()

	group1 := "test-group-1"
	group2 := "test-group-2"

	service := EventBridgeScheduleGroup{Client: mockedEventBridgeScheduleGroupService{
		ListScheduleGroupsOutput: scheduler.ListScheduleGroupsOutput{
			ScheduleGroups: []types.ScheduleGroupSummary{
				{
					Name:         aws.String(group1),
					State:        types.ScheduleGroupStateActive,
					CreationDate: &now,
				},
				{
					Name:         aws.String(group2),
					State:        types.ScheduleGroupStateActive,
					CreationDate: aws.Time(now.Add(time.Hour)),
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
			expected:  []string{group1, group2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(group1),
					}},
				}},
			expected: []string{group2},
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
				config.Config{EventBridgeScheduleGroup: tc.configObj},
			)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(buses))
		})
	}
}

func Test_EventBridgeScheduleGroup_NukeAll(t *testing.T) {
	t.Parallel()

	groupName := "test-group"
	service := EventBridgeScheduleGroup{Client: mockedEventBridgeScheduleGroupService{
		DeleteScheduleGroupOutput: scheduler.DeleteScheduleGroupOutput{},
	}}

	err := service.nukeAll([]*string{&groupName})
	assert.NoError(t, err)
}
