package resources

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

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
	CreationTimes    map[string]time.Time
	DescribeCalls    int
}

func (m *mockDataSyncTaskClient) DeleteTask(ctx context.Context, params *datasync.DeleteTaskInput, optFns ...func(*datasync.Options)) (*datasync.DeleteTaskOutput, error) {
	return &m.DeleteTaskOutput, nil
}

func (m *mockDataSyncTaskClient) DescribeTask(ctx context.Context, params *datasync.DescribeTaskInput, optFns ...func(*datasync.Options)) (*datasync.DescribeTaskOutput, error) {
	m.DescribeCalls++
	createdAt, ok := m.CreationTimes[aws.ToString(params.TaskArn)]
	if !ok {
		return nil, errors.New("access denied")
	}
	return &datasync.DescribeTaskOutput{CreationTime: aws.Time(createdAt)}, nil
}

func (m *mockDataSyncTaskClient) ListTasks(ctx context.Context, params *datasync.ListTasksInput, optFns ...func(*datasync.Options)) (*datasync.ListTasksOutput, error) {
	return &m.ListTasksOutput, nil
}

func (m *mockDataSyncTaskClient) ListTagsForResource(ctx context.Context, params *datasync.ListTagsForResourceInput, optFns ...func(*datasync.Options)) (*datasync.ListTagsForResourceOutput, error) {
	if aws.ToString(params.ResourceArn) == "arn:aws:datasync:us-east-1:123456789012:task/task-1" {
		return &datasync.ListTagsForResourceOutput{
			Tags: []types.TagListEntry{{Key: aws.String("env"), Value: aws.String("prod")}},
		}, nil
	}
	return &datasync.ListTagsForResourceOutput{}, nil
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
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"env": {RE: *regexp.MustCompile("^prod$")}},
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

func TestListDataSyncTasks_TimeFilter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	oldArn := "arn:aws:datasync:us-east-1:123456789012:task/task-old"
	newArn := "arn:aws:datasync:us-east-1:123456789012:task/task-new"
	unreadableArn := "arn:aws:datasync:us-east-1:123456789012:task/task-unreadable"
	newMock := func() *mockDataSyncTaskClient {
		return &mockDataSyncTaskClient{
			ListTasksOutput: datasync.ListTasksOutput{
				Tasks: []types.TaskListEntry{
					{Name: aws.String("old"), TaskArn: aws.String(oldArn)},
					{Name: aws.String("new"), TaskArn: aws.String(newArn)},
					{Name: aws.String("unreadable"), TaskArn: aws.String(unreadableArn)},
				},
			},
			CreationTimes: map[string]time.Time{
				oldArn: now.Add(-2 * time.Hour),
				newArn: now,
			},
		}
	}

	mock := newMock()
	olderThanAnHour := config.ResourceType{ExcludeRule: config.FilterRule{TimeAfter: aws.Time(now.Add(-time.Hour))}}
	arns, err := listDataSyncTasks(context.Background(), mock, resource.Scope{}, olderThanAnHour)
	require.NoError(t, err)
	require.Equal(t, []string{oldArn}, aws.ToStringSlice(arns), "an unreadable creation time leaves the task out")

	mock = newMock()
	arns, err = listDataSyncTasks(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.Len(t, arns, 3)
	require.Zero(t, mock.DescribeCalls, "no time filter, no DescribeTask calls")
}

func TestDeleteDataSyncTask(t *testing.T) {
	t.Parallel()

	mock := &mockDataSyncTaskClient{}
	err := deleteDataSyncTask(context.Background(), mock, aws.String("arn:aws:datasync:us-east-1:123456789012:task/task-1"))
	require.NoError(t, err)
}
