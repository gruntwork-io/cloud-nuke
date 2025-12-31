package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestKinesisFirehose_ResourceName(t *testing.T) {
	r := NewKinesisFirehose()
	assert.Equal(t, "kinesis-firehose", r.ResourceName())
}

func TestKinesisFirehose_MaxBatchSize(t *testing.T) {
	r := NewKinesisFirehose()
	assert.Equal(t, 35, r.MaxBatchSize())
}

func TestListKinesisFirehose(t *testing.T) {
	t.Parallel()

	mock := &mockKinesisFirehoseClient{
		ListDeliveryStreamsOutput: firehose.ListDeliveryStreamsOutput{
			DeliveryStreamNames: []string{"stream1", "stream2"},
		},
	}

	names, err := listKinesisFirehose(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"stream1", "stream2"}, aws.ToStringSlice(names))
}

func TestListKinesisFirehose_WithFilter(t *testing.T) {
	t.Parallel()

	mock := &mockKinesisFirehoseClient{
		ListDeliveryStreamsOutput: firehose.ListDeliveryStreamsOutput{
			DeliveryStreamNames: []string{"stream1", "skip-this"},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
		},
	}

	names, err := listKinesisFirehose(context.Background(), mock, resource.Scope{}, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"stream1"}, aws.ToStringSlice(names))
}

func TestDeleteKinesisFirehose(t *testing.T) {
	t.Parallel()

	mock := &mockKinesisFirehoseClient{}
	err := deleteKinesisFirehose(context.Background(), mock, aws.String("test-stream"))
	require.NoError(t, err)
}
