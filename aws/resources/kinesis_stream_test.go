package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	kinesistypes "github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockKinesisStreamsClient struct {
	ListStreamsOutput            kinesis.ListStreamsOutput
	DeleteStreamOutput          kinesis.DeleteStreamOutput
	DeleteStreamInput           *kinesis.DeleteStreamInput
	DeleteStreamEnforceConsumer *bool
	TagsByStream                map[string][]kinesistypes.Tag
}

func (m *mockKinesisStreamsClient) ListStreams(ctx context.Context, params *kinesis.ListStreamsInput, optFns ...func(*kinesis.Options)) (*kinesis.ListStreamsOutput, error) {
	return &m.ListStreamsOutput, nil
}

func (m *mockKinesisStreamsClient) DeleteStream(ctx context.Context, params *kinesis.DeleteStreamInput, optFns ...func(*kinesis.Options)) (*kinesis.DeleteStreamOutput, error) {
	m.DeleteStreamInput = params
	m.DeleteStreamEnforceConsumer = params.EnforceConsumerDeletion
	return &m.DeleteStreamOutput, nil
}

func (m *mockKinesisStreamsClient) ListTagsForStream(ctx context.Context, params *kinesis.ListTagsForStreamInput, optFns ...func(*kinesis.Options)) (*kinesis.ListTagsForStreamOutput, error) {
	return &kinesis.ListTagsForStreamOutput{Tags: m.TagsByStream[aws.ToString(params.StreamName)]}, nil
}

func TestKinesisStreams_GetAll(t *testing.T) {
	t.Parallel()

	mock := &mockKinesisStreamsClient{
		ListStreamsOutput: kinesis.ListStreamsOutput{
			StreamNames: []string{"stream-keep", "stream-skip"},
		},
		TagsByStream: map[string][]kinesistypes.Tag{
			"stream-keep": {{Key: aws.String("env"), Value: aws.String("prod")}},
			"stream-skip": {{Key: aws.String("env"), Value: aws.String("dev")}},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{"stream-keep", "stream-skip"},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("stream-skip")}},
				},
			},
			expected: []string{"stream-keep"},
		},
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"env": {RE: *regexp.MustCompile("^prod$")},
					},
				},
			},
			expected: []string{"stream-keep"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listKinesisStreams(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestKinesisStreams_NukeAll(t *testing.T) {
	t.Parallel()

	mock := &mockKinesisStreamsClient{}

	err := deleteKinesisStream(context.Background(), mock, aws.String("test-stream"))
	require.NoError(t, err)

	// Verify EnforceConsumerDeletion is set to true
	require.NotNil(t, mock.DeleteStreamEnforceConsumer)
	require.True(t, *mock.DeleteStreamEnforceConsumer, "EnforceConsumerDeletion should be true to handle streams with consumers")
}
