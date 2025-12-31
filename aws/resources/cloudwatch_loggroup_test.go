package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCloudWatchLogGroupsClient struct {
	DescribeLogGroupsOutput cloudwatchlogs.DescribeLogGroupsOutput
	DeleteLogGroupOutput    cloudwatchlogs.DeleteLogGroupOutput
}

func (m *mockCloudWatchLogGroupsClient) DescribeLogGroups(ctx context.Context, params *cloudwatchlogs.DescribeLogGroupsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	return &m.DescribeLogGroupsOutput, nil
}

func (m *mockCloudWatchLogGroupsClient) DeleteLogGroup(ctx context.Context, params *cloudwatchlogs.DeleteLogGroupInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DeleteLogGroupOutput, error) {
	return &m.DeleteLogGroupOutput, nil
}

func TestCloudWatchLogGroups_ResourceName(t *testing.T) {
	r := NewCloudWatchLogGroups()
	assert.Equal(t, "cloudwatch-loggroup", r.ResourceName())
}

func TestCloudWatchLogGroups_MaxBatchSize(t *testing.T) {
	r := NewCloudWatchLogGroups()
	assert.Equal(t, 35, r.MaxBatchSize())
}

func TestListCloudWatchLogGroups(t *testing.T) {
	t.Parallel()

	now := time.Now().UnixMilli()
	mock := &mockCloudWatchLogGroupsClient{
		DescribeLogGroupsOutput: cloudwatchlogs.DescribeLogGroupsOutput{
			LogGroups: []types.LogGroup{
				{LogGroupName: aws.String("log-group-1"), CreationTime: aws.Int64(now)},
				{LogGroupName: aws.String("log-group-2"), CreationTime: aws.Int64(now)},
			},
		},
	}

	names, err := listCloudWatchLogGroups(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"log-group-1", "log-group-2"}, aws.ToStringSlice(names))
}

func TestListCloudWatchLogGroups_WithFilter(t *testing.T) {
	t.Parallel()

	now := time.Now().UnixMilli()
	mock := &mockCloudWatchLogGroupsClient{
		DescribeLogGroupsOutput: cloudwatchlogs.DescribeLogGroupsOutput{
			LogGroups: []types.LogGroup{
				{LogGroupName: aws.String("log-group-1"), CreationTime: aws.Int64(now)},
				{LogGroupName: aws.String("skip-this"), CreationTime: aws.Int64(now)},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
		},
	}

	names, err := listCloudWatchLogGroups(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"log-group-1"}, aws.ToStringSlice(names))
}

func TestDeleteCloudWatchLogGroup(t *testing.T) {
	t.Parallel()

	mock := &mockCloudWatchLogGroupsClient{}
	err := deleteCloudWatchLogGroup(context.Background(), mock, aws.String("test-log-group"))
	require.NoError(t, err)
}
