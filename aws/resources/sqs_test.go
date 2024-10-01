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
	"github.com/stretchr/testify/require"
)

type mockedSQSService struct {
	SqsQueueAPI
	GetQueueAttributesOutput map[string]sqs.GetQueueAttributesOutput
	ListQueuesOutput         sqs.ListQueuesOutput
	DeleteQueueOutput        sqs.DeleteQueueOutput
}

func (m mockedSQSService) GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error) {
	url := params.QueueUrl
	resp := m.GetQueueAttributesOutput[*url]

	return &resp, nil
}

func (m mockedSQSService) ListQueues(context.Context, *sqs.ListQueuesInput, ...func(*sqs.Options)) (*sqs.ListQueuesOutput, error) {
	return &m.ListQueuesOutput, nil
}

func (m mockedSQSService) DeleteQueue(ctx context.Context, params *sqs.DeleteQueueInput, optFns ...func(*sqs.Options)) (*sqs.DeleteQueueOutput, error) {
	return &m.DeleteQueueOutput, nil
}

func TestSqsQueue_GetAll(t *testing.T) {
	t.Parallel()

	queue1 := "https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue1"
	queue2 := "https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue2"
	now := time.Now()
	sq := SqsQueue{
		Client: mockedSQSService{
			ListQueuesOutput: sqs.ListQueuesOutput{
				QueueUrls: []string{
					queue1,
					queue2,
				},
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
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{queue1, queue2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile("MyQueue1"),
					}}},
			},
			expected: []string{queue2},
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
			names, err := sq.getAll(context.Background(), config.Config{
				SQS: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestSqsQueue_NukeAll(t *testing.T) {
	t.Parallel()

	sq := SqsQueue{
		Client: mockedSQSService{
			DeleteQueueOutput: sqs.DeleteQueueOutput{},
		},
	}

	err := sq.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
