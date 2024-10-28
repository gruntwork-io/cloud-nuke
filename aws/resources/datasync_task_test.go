package resources

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/datasync"
	"github.com/aws/aws-sdk-go-v2/service/datasync/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDataSyncTask struct {
	DataSyncTaskAPI
	DeleteTaskOutput datasync.DeleteTaskOutput
	ListTasksOutput  datasync.ListTasksOutput
}

func (m mockDataSyncTask) DeleteTask(ctx context.Context, params *datasync.DeleteTaskInput, optFns ...func(*datasync.Options)) (*datasync.DeleteTaskOutput, error) {
	return &m.DeleteTaskOutput, nil
}

func (m mockDataSyncTask) ListTasks(ctx context.Context, params *datasync.ListTasksInput, optFns ...func(*datasync.Options)) (*datasync.ListTasksOutput, error) {
	return &m.ListTasksOutput, nil
}

func TestDataSyncTask_NukeAll(t *testing.T) {
	t.Parallel()

	testName := "test-datasync-task"
	task := DataSyncTask{
		Client: mockDataSyncTask{
			DeleteTaskOutput: datasync.DeleteTaskOutput{},
		},
	}

	err := task.nukeAll([]*string{&testName})
	assert.NoError(t, err)
}

func TestDataSyncTask_GetAll(t *testing.T) {
	t.Parallel()

	testName1 := "test-task-1"
	testName2 := "test-task-2"

	task := DataSyncTask{Client: mockDataSyncTask{
		ListTasksOutput: datasync.ListTasksOutput{
			Tasks: []types.TaskListEntry{
				{
					Name:    &testName1,
					TaskArn: aws.String(fmt.Sprintf("arn::%s", testName1)),
				},
				{
					Name:    &testName2,
					TaskArn: aws.String(fmt.Sprintf("arn::%s", testName2)),
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
			expected:  []string{fmt.Sprintf("arn::%s", testName1), fmt.Sprintf("arn::%s", testName2)},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}},
				}},
			expected: []string{fmt.Sprintf("arn::%s", testName2)},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := task.getAll(context.Background(), config.Config{
				DataSyncTask: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}
