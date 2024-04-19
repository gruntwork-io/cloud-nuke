package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"regexp"
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedSNSTopic struct {
	snsiface.SNSAPI
	ListTopicsOutput             sns.ListTopicsOutput
	ListTagsForResourceOutputMap map[string]sns.ListTagsForResourceOutput
	DeleteTopicOutput            sns.DeleteTopicOutput
}

func (m mockedSNSTopic) ListTopicsPages(input *sns.ListTopicsInput, fn func(*sns.ListTopicsOutput, bool) bool) error {
	fn(&m.ListTopicsOutput, true)
	return nil
}

func (m mockedSNSTopic) ListTagsForResource(input *sns.ListTagsForResourceInput) (*sns.ListTagsForResourceOutput, error) {
	arn := input.ResourceArn
	resp := m.ListTagsForResourceOutputMap[*arn]

	return &resp, nil
}

func (m mockedSNSTopic) DeleteTopic(input *sns.DeleteTopicInput) (*sns.DeleteTopicOutput, error) {
	return &m.DeleteTopicOutput, nil
}

func TestSNS_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testTopic1 := "arn:aws:sns:us-east-1:123456789012:MyTopic1"
	testTopic2 := "arn:aws:sns:us-east-1:123456789012:MyTopic2"
	now := time.Now()
	s := SNSTopic{
		Client: mockedSNSTopic{
			ListTopicsOutput: sns.ListTopicsOutput{
				Topics: []*sns.Topic{
					{TopicArn: awsgo.String(testTopic1)},
					{TopicArn: awsgo.String(testTopic2)},
				},
			},
			ListTagsForResourceOutputMap: map[string]sns.ListTagsForResourceOutput{
				testTopic1: {
					Tags: []*sns.Tag{{
						Key:   awsgo.String(firstSeenTagKey),
						Value: awsgo.String(now.Format(firstSeenTimeFormat)),
					}},
				},
				testTopic2: {
					Tags: []*sns.Tag{{
						Key:   awsgo.String(firstSeenTagKey),
						Value: awsgo.String(now.Add(1).Format(firstSeenTimeFormat)),
					}},
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
			expected:  []string{testTopic1, testTopic2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile("MyTopic1"),
					}}},
			},
			expected: []string{testTopic2},
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
			names, err := s.getAll(context.Background(), config.Config{
				SNS: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestSNS_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	s := SNSTopic{
		Client: mockedSNSTopic{
			DeleteTopicOutput: sns.DeleteTopicOutput{},
		},
	}

	err := s.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
