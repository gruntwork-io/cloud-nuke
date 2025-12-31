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
	testArn1 := fmt.Sprintf("arn::%s", testName1)
	testArn2 := fmt.Sprintf("arn::%s", testName2)

	mock := &mockDataSyncTaskClient{
		ListTasksOutput: datasync.ListTasksOutput{
			Tasks: []types.TaskListEntry{
				{Name: &testName1, TaskArn: aws.String(testArn1)},
				{Name: &testName2, TaskArn: aws.String(testArn2)},
			},
		},
	}

	arns, err := listDataSyncTasks(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{testArn1, testArn2}, aws.ToStringSlice(arns))
}

func TestListDataSyncTasks_WithFilter(t *testing.T) {
	t.Parallel()

	testName1 := "test-task-1"
	testName2 := "skip-this"
	testArn1 := fmt.Sprintf("arn::%s", testName1)
	testArn2 := fmt.Sprintf("arn::%s", testName2)

	mock := &mockDataSyncTaskClient{
		ListTasksOutput: datasync.ListTasksOutput{
			Tasks: []types.TaskListEntry{
				{Name: &testName1, TaskArn: aws.String(testArn1)},
				{Name: &testName2, TaskArn: aws.String(testArn2)},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
		},
	}

	arns, err := listDataSyncTasks(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{testArn1}, aws.ToStringSlice(arns))
}

func TestDeleteDataSyncTask(t *testing.T) {
	t.Parallel()

	mock := &mockDataSyncTaskClient{}
	err := deleteDataSyncTask(context.Background(), mock, aws.String("arn::test-task"))
	require.NoError(t, err)
}
