package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockKinesisStreamsClient struct {
	ListStreamsOutput  kinesis.ListStreamsOutput
	DeleteStreamOutput kinesis.DeleteStreamOutput
}

func (m *mockKinesisStreamsClient) ListStreams(ctx context.Context, params *kinesis.ListStreamsInput, optFns ...func(*kinesis.Options)) (*kinesis.ListStreamsOutput, error) {
	return &m.ListStreamsOutput, nil
}

func (m *mockKinesisStreamsClient) DeleteStream(ctx context.Context, params *kinesis.DeleteStreamInput, optFns ...func(*kinesis.Options)) (*kinesis.DeleteStreamOutput, error) {
	return &m.DeleteStreamOutput, nil
}

func TestKinesisStreams_ResourceName(t *testing.T) {
	r := NewKinesisStreams()
	assert.Equal(t, "kinesis-stream", r.ResourceName())
}

func TestKinesisStreams_MaxBatchSize(t *testing.T) {
	r := NewKinesisStreams()
	assert.Equal(t, 35, r.MaxBatchSize())
}

func TestListKinesisStreams(t *testing.T) {
	t.Parallel()

	mock := &mockKinesisStreamsClient{
		ListStreamsOutput: kinesis.ListStreamsOutput{
			StreamNames: []string{"stream1", "stream2"},
		},
	}

	names, err := listKinesisStreams(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"stream1", "stream2"}, aws.ToStringSlice(names))
}

func TestListKinesisStreams_WithFilter(t *testing.T) {
	t.Parallel()

	mock := &mockKinesisStreamsClient{
		ListStreamsOutput: kinesis.ListStreamsOutput{
			StreamNames: []string{"stream1", "skip-this"},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
		},
	}

	names, err := listKinesisStreams(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"stream1"}, aws.ToStringSlice(names))
}

func TestDeleteKinesisStream(t *testing.T) {
	t.Parallel()

	mock := &mockKinesisStreamsClient{}
	err := deleteKinesisStream(context.Background(), mock, aws.String("test-stream"))
	require.NoError(t, err)
}
