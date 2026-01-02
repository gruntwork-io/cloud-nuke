package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

// mockKinesisFirehoseClient implements KinesisFirehoseAPI for testing.
type mockKinesisFirehoseClient struct {
	ListDeliveryStreamsOutput  firehose.ListDeliveryStreamsOutput
	DeleteDeliveryStreamOutput firehose.DeleteDeliveryStreamOutput
}

func (m *mockKinesisFirehoseClient) ListDeliveryStreams(ctx context.Context, params *firehose.ListDeliveryStreamsInput, optFns ...func(*firehose.Options)) (*firehose.ListDeliveryStreamsOutput, error) {
	return &m.ListDeliveryStreamsOutput, nil
}

func (m *mockKinesisFirehoseClient) DeleteDeliveryStream(ctx context.Context, params *firehose.DeleteDeliveryStreamInput, optFns ...func(*firehose.Options)) (*firehose.DeleteDeliveryStreamOutput, error) {
	return &m.DeleteDeliveryStreamOutput, nil
}

func TestListKinesisFirehose(t *testing.T) {
	t.Parallel()

	mock := &mockKinesisFirehoseClient{
		ListDeliveryStreamsOutput: firehose.ListDeliveryStreamsOutput{
			DeliveryStreamNames: []string{"stream1", "stream2"},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{"stream1", "stream2"},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile("stream1"),
					}},
				},
			},
			expected: []string{"stream2"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listKinesisFirehose(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteKinesisFirehose(t *testing.T) {
	t.Parallel()

	mock := &mockKinesisFirehoseClient{}
	err := deleteKinesisFirehose(context.Background(), mock, aws.String("test-stream"))
	require.NoError(t, err)
}
