package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

// mockKinesisStreamsClient implements KinesisStreamsAPI for testing.
type mockKinesisStreamsClient struct {
	ListStreamsOutput           kinesis.ListStreamsOutput
	DeleteStreamOutput          kinesis.DeleteStreamOutput
	DeleteStreamInput           *kinesis.DeleteStreamInput // Captures input for verification
	DeleteStreamEnforceConsumer *bool
}

func (m *mockKinesisStreamsClient) ListStreams(ctx context.Context, params *kinesis.ListStreamsInput, optFns ...func(*kinesis.Options)) (*kinesis.ListStreamsOutput, error) {
	return &m.ListStreamsOutput, nil
}

func (m *mockKinesisStreamsClient) DeleteStream(ctx context.Context, params *kinesis.DeleteStreamInput, optFns ...func(*kinesis.Options)) (*kinesis.DeleteStreamOutput, error) {
	m.DeleteStreamInput = params
	m.DeleteStreamEnforceConsumer = params.EnforceConsumerDeletion
	return &m.DeleteStreamOutput, nil
}

func TestKinesisStreams_GetAll(t *testing.T) {
	t.Parallel()

	mock := &mockKinesisStreamsClient{
		ListStreamsOutput: kinesis.ListStreamsOutput{
			StreamNames: []string{"stream-keep", "stream-skip"},
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
