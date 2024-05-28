package resources

import (
	"context"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedSqsQueue struct {
	sqsiface.SQSAPI
	DeleteQueueOutput        sqs.DeleteQueueOutput
	GetQueueAttributesOutput map[string]sqs.GetQueueAttributesOutput
	ListQueuesOutput         sqs.ListQueuesOutput
}

func (m mockedSqsQueue) ListQueuesPagesWithContext(_ awsgo.Context, _ *sqs.ListQueuesInput, fn func(*sqs.ListQueuesOutput, bool) bool, _ ...request.Option) error {
	fn(&m.ListQueuesOutput, true)
	return nil
}

func (m mockedSqsQueue) GetQueueAttributesWithContext(_ awsgo.Context, input *sqs.GetQueueAttributesInput, _ ...request.Option) (*sqs.GetQueueAttributesOutput, error) {
	url := input.QueueUrl
	resp := m.GetQueueAttributesOutput[*url]

	return &resp, nil
}

func (m mockedSqsQueue) DeleteQueueWithContext(_ awsgo.Context, _ *sqs.DeleteQueueInput, _ ...request.Option) (*sqs.DeleteQueueOutput, error) {
	return &m.DeleteQueueOutput, nil
}

func TestSqsQueue_GetAll(t *testing.T) {

	t.Parallel()

	queue1 := "https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue1"
	queue2 := "https://sqs.us-east-1.amazonaws.com/123456789012/MyQueue2"
	now := time.Now()
	sq := SqsQueue{
		Client: mockedSqsQueue{
			ListQueuesOutput: sqs.ListQueuesOutput{
				QueueUrls: []*string{
					awsgo.String(queue1),
					awsgo.String(queue2),
				},
			},
			GetQueueAttributesOutput: map[string]sqs.GetQueueAttributesOutput{
				queue1: {
					Attributes: map[string]*string{
						"CreatedTimestamp": awsgo.String(strconv.FormatInt(now.Unix(), 10)),
					},
				},
				queue2: {
					Attributes: map[string]*string{
						"CreatedTimestamp": awsgo.String(strconv.FormatInt(now.Add(1).Unix(), 10)),
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
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}
}

func TestSqsQueue_NukeAll(t *testing.T) {

	t.Parallel()

	sq := SqsQueue{
		Client: mockedSqsQueue{
			DeleteQueueOutput: sqs.DeleteQueueOutput{},
		},
	}

	err := sq.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
