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
	ListQueuesOutput         sqs.ListQueuesOutput
	GetQueueAttributesOutput map[string]sqs.GetQueueAttributesOutput
	DeleteQueueOutput        sqs.DeleteQueueOutput
}

func (m *mockSqsQueueClient) ListQueues(ctx context.Context, params *sqs.ListQueuesInput, optFns ...func(*sqs.Options)) (*sqs.ListQueuesOutput, error) {
	return &m.ListQueuesOutput, nil
}

func (m *mockSqsQueueClient) GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error) {
	url := aws.ToString(params.QueueUrl)
	resp := m.GetQueueAttributesOutput[url]
	return &resp, nil
}

func (m *mockSqsQueueClient) DeleteQueue(ctx context.Context, params *sqs.DeleteQueueInput, optFns ...func(*sqs.Options)) (*sqs.DeleteQueueOutput, error) {
	return &m.DeleteQueueOutput, nil
}

func TestListSqsQueues(t *testing.T) {
	t.Parallel()

	now := time.Now()
	queue1 := "https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue1"
	queue2 := "https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue2"

	mock := &mockSqsQueueClient{
		ListQueuesOutput: sqs.ListQueuesOutput{
			QueueUrls: []string{queue1, queue2},
		},
		GetQueueAttributesOutput: map[string]sqs.GetQueueAttributesOutput{
			queue1: {Attributes: map[string]string{"CreatedTimestamp": strconv.FormatInt(now.Unix(), 10)}},
			queue2: {Attributes: map[string]string{"CreatedTimestamp": strconv.FormatInt(now.Unix(), 10)}},
		},
	}

	tests := map[string]struct {
		cfg      config.ResourceType
		expected []string
	}{
		"no filter": {
			cfg:      config.ResourceType{},
			expected: []string{queue1, queue2},
		},
		"name exclusion filter": {
			cfg: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("MyQueue1")}},
				},
			},
			expected: []string{queue2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			urls, err := listSqsQueues(context.Background(), mock, resource.Scope{}, tc.cfg)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(urls))
		})
	}
}

func TestDeleteSqsQueue(t *testing.T) {
	t.Parallel()

	mock := &mockSqsQueueClient{}
	err := deleteSqsQueue(context.Background(), mock, aws.String("https://sqs.us-east-1.amazonaws.com/123456789012/TestQueue"))
	require.NoError(t, err)
}
