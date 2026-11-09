package resources

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	kinesistypes "github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockKinesisStreamsClient struct {
	ListStreamsOutput           kinesis.ListStreamsOutput
	DeleteStreamOutput          kinesis.DeleteStreamOutput
	DeleteStreamInput           *kinesis.DeleteStreamInput
	DeleteStreamEnforceConsumer *bool
	TagsByStream                map[string][]kinesistypes.Tag
	// CreatedByStream answers DescribeStreamSummary.
	CreatedByStream map[string]time.Time
	// DescribedStreams records the streams passed to DescribeStreamSummary.
	DescribedStreams []string
}

// DescribeStreamSummary records the stream and returns its CreatedByStream time.
func (m *mockKinesisStreamsClient) DescribeStreamSummary(ctx context.Context, params *kinesis.DescribeStreamSummaryInput, optFns ...func(*kinesis.Options)) (*kinesis.DescribeStreamSummaryOutput, error) {
	name := aws.ToString(params.StreamName)
	m.DescribedStreams = append(m.DescribedStreams, name)
	created, ok := m.CreatedByStream[name]
	if !ok {
		return nil, errors.New("ResourceNotFoundException")
	}
	return &kinesis.DescribeStreamSummaryOutput{
		StreamDescriptionSummary: &kinesistypes.StreamDescriptionSummary{StreamCreationTimestamp: aws.Time(created)},
	}, nil
}

// ListStreams returns ListStreamsOutput.
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

// TestKinesisStreams_GetAll checks the name, tag and time filters on listed streams.
func TestKinesisStreams_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockKinesisStreamsClient{
		ListStreamsOutput: kinesis.ListStreamsOutput{
			StreamNames: []string{"stream-keep", "stream-skip"},
			StreamSummaries: []kinesistypes.StreamSummary{
				{StreamName: aws.String("stream-keep"), StreamCreationTimestamp: aws.Time(now.Add(-2 * time.Hour))},
				{StreamName: aws.String("stream-skip"), StreamCreationTimestamp: aws.Time(now.Add(-10 * time.Minute))},
			},
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
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{TimeAfter: aws.Time(now.Add(-1 * time.Hour))},
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

// TestKinesisStreams_GetAll_StreamWithoutSummary checks that a stream with no
// summary gets its creation time from DescribeStreamSummary, only under a time filter.
func TestKinesisStreams_GetAll_StreamWithoutSummary(t *testing.T) {
	t.Parallel()

	now := time.Now()
	newMock := func() *mockKinesisStreamsClient {
		return &mockKinesisStreamsClient{
			ListStreamsOutput: kinesis.ListStreamsOutput{
				StreamNames: []string{"old-unsummarized", "unreadable"},
			},
			CreatedByStream: map[string]time.Time{"old-unsummarized": now.Add(-2 * time.Hour)},
		}
	}
	olderThanOneHour := config.ResourceType{
		ExcludeRule: config.FilterRule{TimeAfter: aws.Time(now.Add(-1 * time.Hour))},
	}

	mock := newMock()
	names, err := listKinesisStreams(context.Background(), mock, resource.Scope{}, olderThanOneHour)
	require.NoError(t, err)
	require.Equal(t, []string{"old-unsummarized"}, aws.ToStringSlice(names))
	require.Equal(t, []string{"old-unsummarized", "unreadable"}, mock.DescribedStreams)

	mock = newMock()
	names, err = listKinesisStreams(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.Equal(t, []string{"old-unsummarized", "unreadable"}, aws.ToStringSlice(names))
	require.Empty(t, mock.DescribedStreams, "no time filter, so no extra call")
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
