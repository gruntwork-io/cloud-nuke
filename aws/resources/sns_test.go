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

type mockSNSClient struct {
	ListTopicsOutput          sns.ListTopicsOutput
	ListTagsForResourceOutput map[string]sns.ListTagsForResourceOutput
	TagResourceOutput         sns.TagResourceOutput
	DeleteTopicOutput         sns.DeleteTopicOutput
}

func (m *mockSNSClient) ListTopics(ctx context.Context, params *sns.ListTopicsInput, optFns ...func(*sns.Options)) (*sns.ListTopicsOutput, error) {
	return &m.ListTopicsOutput, nil
}

func (m *mockSNSClient) ListTagsForResource(ctx context.Context, params *sns.ListTagsForResourceInput, optFns ...func(*sns.Options)) (*sns.ListTagsForResourceOutput, error) {
	if m.ListTagsForResourceOutput != nil {
		resp := m.ListTagsForResourceOutput[*params.ResourceArn]
		return &resp, nil
	}
	return &sns.ListTagsForResourceOutput{}, nil
}

func (m *mockSNSClient) TagResource(ctx context.Context, params *sns.TagResourceInput, optFns ...func(*sns.Options)) (*sns.TagResourceOutput, error) {
	return &m.TagResourceOutput, nil
}

func (m *mockSNSClient) DeleteTopic(ctx context.Context, params *sns.DeleteTopicInput, optFns ...func(*sns.Options)) (*sns.DeleteTopicOutput, error) {
	return &m.DeleteTopicOutput, nil
}

func TestListSNSTopics(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testTopic1 := "arn:aws:sns:us-east-1:123456789012:MyTopic1"
	testTopic2 := "arn:aws:sns:us-east-1:123456789012:MyTopic2"

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
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("MyTopic1")}},
				},
			},
			expected: []string{testTopic2},
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

			mock := &mockSNSClient{
				ListTopicsOutput: sns.ListTopicsOutput{
					Topics: []types.Topic{
						{TopicArn: aws.String(testTopic1)},
						{TopicArn: aws.String(testTopic2)},
					},
				},
				ListTagsForResourceOutput: map[string]sns.ListTagsForResourceOutput{
					testTopic1: {
						Tags: []types.Tag{{
							Key:   aws.String(util.FirstSeenTagKey),
							Value: aws.String(util.FormatTimestamp(now)),
						}},
					},
					testTopic2: {
						Tags: []types.Tag{{
							Key:   aws.String(util.FirstSeenTagKey),
							Value: aws.String(util.FormatTimestamp(now)),
						}},
					},
				},
			}

			names, err := listSNSTopics(ctx, mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteSNSTopic(t *testing.T) {
	t.Parallel()

	mock := &mockSNSClient{}
	err := deleteSNSTopic(context.Background(), mock, aws.String("arn:aws:sns:us-east-1:123456789012:TestTopic"))
	require.NoError(t, err)
}
