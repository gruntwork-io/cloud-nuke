package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedSNSService struct {
	SNSTopic
	DeleteTopicOutput         sns.DeleteTopicOutput
	TagResourceOutput         sns.TagResourceOutput
	ListTopicsOutput          sns.ListTopicsOutput
	ListTagsForResourceOutput map[string]sns.ListTagsForResourceOutput
}

func (m mockedSNSService) ListTopics(context.Context, *sns.ListTopicsInput, ...func(*sns.Options)) (*sns.ListTopicsOutput, error) {
	return &m.ListTopicsOutput, nil
}

func (m mockedSNSService) ListTagsForResource(ctx context.Context, params *sns.ListTagsForResourceInput, optFns ...func(*sns.Options)) (*sns.ListTagsForResourceOutput, error) {
	arn := params.ResourceArn
	resp := m.ListTagsForResourceOutput[*arn]

	return &resp, nil
}

func (m mockedSNSService) TagResource(ctx context.Context, params *sns.TagResourceInput, optFns ...func(*sns.Options)) (*sns.TagResourceOutput, error) {
	return &m.TagResourceOutput, nil
}

func (m mockedSNSService) DeleteTopic(ctx context.Context, params *sns.DeleteTopicInput, optFns ...func(*sns.Options)) (*sns.DeleteTopicOutput, error) {
	return &m.DeleteTopicOutput, nil
}

func TestSNS_GetAll(t *testing.T) {
	t.Parallel()

	// Set excludeFirstSeenTag to false for testing
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	testTopic1 := "arn:aws:sns:us-east-1:123456789012:MyTopic1"
	testTopic2 := "arn:aws:sns:us-east-1:123456789012:MyTopic2"
	now := time.Now()
	s := SNSTopic{
		Client: mockedSNSService{
			ListTopicsOutput: sns.ListTopicsOutput{
				Topics: []types.Topic{
					{TopicArn: aws.String(testTopic1)},
					{TopicArn: aws.String(testTopic2)},
				},
			},
			ListTagsForResourceOutput: map[string]sns.ListTagsForResourceOutput{
				testTopic1: {
					Tags: []types.Tag{{
						Key:   aws.String(firstSeenTagKey),
						Value: aws.String(now.Format(firstSeenTimeFormat)),
					}},
				},
				testTopic2: {
					Tags: []types.Tag{{
						Key:   aws.String(firstSeenTagKey),
						Value: aws.String(now.Add(1).Format(firstSeenTimeFormat)),
					}},
				},
			},
		},
	}

	tests := map[string]struct {
		ctx       context.Context
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			ctx:       ctx,
			configObj: config.ResourceType{},
			expected:  []string{testTopic1, testTopic2},
		},
		"nameExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile("MyTopic1"),
					}}},
			},
			expected: []string{testTopic2},
		},
		"timeAfterExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := s.getAll(tc.ctx, config.Config{
				SNS: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestSNS_NukeAll(t *testing.T) {
	t.Parallel()

	s := SNSTopic{
		Client: mockedSNSService{
			DeleteTopicOutput: sns.DeleteTopicOutput{},
		},
	}

	err := s.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
