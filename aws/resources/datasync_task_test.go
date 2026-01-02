package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/datasync"
	"github.com/aws/aws-sdk-go-v2/service/datasync/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockDataSyncTaskClient struct {
	DeleteTaskOutput datasync.DeleteTaskOutput
	ListTasksOutput  datasync.ListTasksOutput
}

func (m *mockDataSyncTaskClient) DeleteTask(ctx context.Context, params *datasync.DeleteTaskInput, optFns ...func(*datasync.Options)) (*datasync.DeleteTaskOutput, error) {
	return &m.DeleteTaskOutput, nil
}

func (m *mockDataSyncTaskClient) ListTasks(ctx context.Context, params *datasync.ListTasksInput, optFns ...func(*datasync.Options)) (*datasync.ListTasksOutput, error) {
	return &m.ListTasksOutput, nil
}

func TestListDataSyncTasks(t *testing.T) {
	t.Parallel()

	testName1 := "test-task-1"
	testName2 := "test-task-2"
	testArn1 := "arn:aws:datasync:us-east-1:123456789012:task/task-1"
	testArn2 := "arn:aws:datasync:us-east-1:123456789012:task/task-2"

	mock := &mockDataSyncTaskClient{
		ListTasksOutput: datasync.ListTasksOutput{
			Tasks: []types.TaskListEntry{
				{Name: aws.String(testName1), TaskArn: aws.String(testArn1)},
				{Name: aws.String(testName2), TaskArn: aws.String(testArn2)},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testArn1, testArn2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("task-2")}},
				},
			},
			expected: []string{testArn1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			arns, err := listDataSyncTasks(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(arns))
		})
	}
}

func TestDeleteDataSyncTask(t *testing.T) {
	t.Parallel()

	mock := &mockDataSyncTaskClient{}
	err := deleteDataSyncTask(context.Background(), mock, aws.String("arn:aws:datasync:us-east-1:123456789012:task/task-1"))
	require.NoError(t, err)
}
