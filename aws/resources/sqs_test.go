package resources

import (
	"context"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockSqsQueueClient struct {
	GetQueueAttributesOutput map[string]sqs.GetQueueAttributesOutput
	ListQueuesOutput         sqs.ListQueuesOutput
	DeleteQueueOutput        sqs.DeleteQueueOutput
}

func (m *mockSqsQueueClient) GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error) {
	url := params.QueueUrl
	resp := m.GetQueueAttributesOutput[*url]
	return &resp, nil
}

func (m *mockSqsQueueClient) ListQueues(ctx context.Context, params *sqs.ListQueuesInput, optFns ...func(*sqs.Options)) (*sqs.ListQueuesOutput, error) {
	return &m.ListQueuesOutput, nil
}

func (m *mockSqsQueueClient) DeleteQueue(ctx context.Context, params *sqs.DeleteQueueInput, optFns ...func(*sqs.Options)) (*sqs.DeleteQueueOutput, error) {
	return &m.DeleteQueueOutput, nil
}

func TestListSqsQueues(t *testing.T) {
	t.Parallel()

	queue1 := "https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue1"
	queue2 := "https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue2"
	now := time.Now()

	mock := &mockSqsQueueClient{
		ListQueuesOutput: sqs.ListQueuesOutput{
			QueueUrls: []string{queue1, queue2},
		},
		GetQueueAttributesOutput: map[string]sqs.GetQueueAttributesOutput{
			queue1: {
				Attributes: map[string]string{
					"CreatedTimestamp": strconv.FormatInt(now.Unix(), 10),
				},
			},
			queue2: {
				Attributes: map[string]string{
					"CreatedTimestamp": strconv.FormatInt(now.Add(1).Unix(), 10),
				},
			},
		},
	}

	urls, err := listSqsQueues(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{queue1, queue2}, aws.ToStringSlice(urls))
}

func TestListSqsQueues_WithFilter(t *testing.T) {
	t.Parallel()

	queue1 := "https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue1"
	queue2 := "https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue2"
	now := time.Now()

	mock := &mockSqsQueueClient{
		ListQueuesOutput: sqs.ListQueuesOutput{
			QueueUrls: []string{queue1, queue2},
		},
		GetQueueAttributesOutput: map[string]sqs.GetQueueAttributesOutput{
			queue1: {
				Attributes: map[string]string{
					"CreatedTimestamp": strconv.FormatInt(now.Unix(), 10),
				},
			},
			queue2: {
				Attributes: map[string]string{
					"CreatedTimestamp": strconv.FormatInt(now.Unix(), 10),
				},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("MyQueue1")}},
		},
	}

	urls, err := listSqsQueues(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{queue2}, aws.ToStringSlice(urls))
}

func TestListSqsQueues_TimeFilter(t *testing.T) {
	t.Parallel()

	queue1 := "https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue1"
	now := time.Now()

	mock := &mockSqsQueueClient{
		ListQueuesOutput: sqs.ListQueuesOutput{
			QueueUrls: []string{queue1},
		},
		GetQueueAttributesOutput: map[string]sqs.GetQueueAttributesOutput{
			queue1: {
				Attributes: map[string]string{
					"CreatedTimestamp": strconv.FormatInt(now.Unix(), 10),
				},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
		},
	}

	urls, err := listSqsQueues(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Empty(t, urls)
}

func TestDeleteSqsQueue(t *testing.T) {
	t.Parallel()

	mock := &mockSqsQueueClient{}
	err := deleteSqsQueue(context.Background(), mock, aws.String("https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue"))
	require.NoError(t, err)
}
