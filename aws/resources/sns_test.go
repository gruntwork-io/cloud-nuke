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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedSNSTopic struct {
	SNSTopicAPI
	ListTopicsOutput          sns.ListTopicsOutput
	ListTagsForResourceOutput map[string]sns.ListTagsForResourceOutput
	TagResourceOutput         sns.TagResourceOutput
	DeleteTopicOutput         sns.DeleteTopicOutput
}

func (m mockedSNSTopic) ListTopics(ctx context.Context, params *sns.ListTopicsInput, optFns ...func(*sns.Options)) (*sns.ListTopicsOutput, error) {
	return &m.ListTopicsOutput, nil
}

func (m mockedSNSTopic) ListTagsForResource(ctx context.Context, params *sns.ListTagsForResourceInput, optFns ...func(*sns.Options)) (*sns.ListTagsForResourceOutput, error) {
	if m.ListTagsForResourceOutput != nil {
		resp := m.ListTagsForResourceOutput[*params.ResourceArn]
		return &resp, nil
	}
	return &sns.ListTagsForResourceOutput{}, nil
}

func (m mockedSNSTopic) TagResource(ctx context.Context, params *sns.TagResourceInput, optFns ...func(*sns.Options)) (*sns.TagResourceOutput, error) {
	return &m.TagResourceOutput, nil
}

func (m mockedSNSTopic) DeleteTopic(ctx context.Context, params *sns.DeleteTopicInput, optFns ...func(*sns.Options)) (*sns.DeleteTopicOutput, error) {
	return &m.DeleteTopicOutput, nil
}

func TestSNSTopic_GetAll(t *testing.T) {
	t.Parallel()

	testArn1 := "arn:aws:sns:us-east-1:123456789012:MyTopic1"
	testArn2 := "arn:aws:sns:us-east-1:123456789012:MyTopic2"
	now := time.Now()

	mock := mockedSNSTopic{
		ListTopicsOutput: sns.ListTopicsOutput{
			Topics: []types.Topic{
				{TopicArn: aws.String(testArn1)},
				{TopicArn: aws.String(testArn2)},
			},
		},
		ListTagsForResourceOutput: map[string]sns.ListTagsForResourceOutput{
			testArn1: {
				Tags: []types.Tag{{
					Key:   aws.String(util.FirstSeenTagKey),
					Value: aws.String(util.FormatTimestamp(now)),
				}},
			},
			testArn2: {
				Tags: []types.Tag{{
					Key:   aws.String(util.FirstSeenTagKey),
					Value: aws.String(util.FormatTimestamp(now)),
				}},
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
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("MyTopic1")}},
				},
			},
			expected: []string{testArn2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				},
			},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
			topics, err := listSNSTopics(ctx, mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(topics))
		})
	}
}

func TestSNSTopic_NukeAll(t *testing.T) {
	t.Parallel()

	mock := mockedSNSTopic{
		DeleteTopicOutput: sns.DeleteTopicOutput{},
	}

	err := deleteSNSTopic(context.Background(), mock, aws.String("arn:aws:sns:us-east-1:123456789012:TestTopic"))
	require.NoError(t, err)
}
