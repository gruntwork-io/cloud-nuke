package resources

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

type mockedSqsQueue struct {
	sqsiface.SQSAPI
	DeleteQueueOutput        sqs.DeleteQueueOutput
	GetQueueAttributesOutput map[string]sqs.GetQueueAttributesOutput
	ListQueuesOutput         sqs.ListQueuesOutput
}

func (m mockedSqsQueue) ListQueuesPages(input *sqs.ListQueuesInput, fn func(*sqs.ListQueuesOutput, bool) bool) error {
	fn(&m.ListQueuesOutput, true)
	return nil
}

func (m mockedSqsQueue) GetQueueAttributes(input *sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error) {
	url := input.QueueUrl
	resp := m.GetQueueAttributesOutput[*url]

	return &resp, nil
}

func (m mockedSqsQueue) DeleteQueue(*sqs.DeleteQueueInput) (*sqs.DeleteQueueOutput, error) {
	return &m.DeleteQueueOutput, nil
}

func TestSqsQueue_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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
						"CreatedTimestamp": awsgo.String(now.Format(time.RFC3339)),
					},
				},
				queue2: {
					Attributes: map[string]*string{
						"CreatedTimestamp": awsgo.String(now.Add(1).Format(time.RFC3339)),
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
			names, err := sq.getAll(config.Config{
				SQS: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}
}

func TestSqsQueue_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	sq := SqsQueue{
		Client: mockedSqsQueue{
			DeleteQueueOutput: sqs.DeleteQueueOutput{},
		},
	}

	err := sq.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
