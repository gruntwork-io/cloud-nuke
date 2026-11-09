package resources

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	firehosetypes "github.com/aws/aws-sdk-go-v2/service/firehose/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockKinesisFirehoseClient struct {
	ListDeliveryStreamsOutput  firehose.ListDeliveryStreamsOutput
	DeleteDeliveryStreamOutput firehose.DeleteDeliveryStreamOutput
	TagsByStream               map[string][]firehosetypes.Tag
	CreationTimes              map[string]time.Time
	DescribeCalls              int
}

func (m *mockKinesisFirehoseClient) DescribeDeliveryStream(ctx context.Context, params *firehose.DescribeDeliveryStreamInput, optFns ...func(*firehose.Options)) (*firehose.DescribeDeliveryStreamOutput, error) {
	m.DescribeCalls++
	createdAt, ok := m.CreationTimes[aws.ToString(params.DeliveryStreamName)]
	if !ok {
		return nil, errors.New("access denied")
	}
	return &firehose.DescribeDeliveryStreamOutput{
		DeliveryStreamDescription: &firehosetypes.DeliveryStreamDescription{CreateTimestamp: aws.Time(createdAt)},
	}, nil
}

func (m *mockKinesisFirehoseClient) ListDeliveryStreams(ctx context.Context, params *firehose.ListDeliveryStreamsInput, optFns ...func(*firehose.Options)) (*firehose.ListDeliveryStreamsOutput, error) {
	return &m.ListDeliveryStreamsOutput, nil
}

func (m *mockKinesisFirehoseClient) DeleteDeliveryStream(ctx context.Context, params *firehose.DeleteDeliveryStreamInput, optFns ...func(*firehose.Options)) (*firehose.DeleteDeliveryStreamOutput, error) {
	return &m.DeleteDeliveryStreamOutput, nil
}

func (m *mockKinesisFirehoseClient) ListTagsForDeliveryStream(ctx context.Context, params *firehose.ListTagsForDeliveryStreamInput, optFns ...func(*firehose.Options)) (*firehose.ListTagsForDeliveryStreamOutput, error) {
	return &firehose.ListTagsForDeliveryStreamOutput{Tags: m.TagsByStream[aws.ToString(params.DeliveryStreamName)]}, nil
}

func TestListKinesisFirehose(t *testing.T) {
	t.Parallel()

	mock := &mockKinesisFirehoseClient{
		ListDeliveryStreamsOutput: firehose.ListDeliveryStreamsOutput{
			DeliveryStreamNames: []string{"stream1", "stream2"},
		},
		TagsByStream: map[string][]firehosetypes.Tag{
			"stream1": {{Key: aws.String("env"), Value: aws.String("prod")}},
			"stream2": {{Key: aws.String("env"), Value: aws.String("dev")}},
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
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"env": {RE: *regexp.MustCompile("^prod$")},
					},
				},
			},
			expected: []string{"stream1"},
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

func TestListKinesisFirehose_TimeFilter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	newMock := func() *mockKinesisFirehoseClient {
		return &mockKinesisFirehoseClient{
			ListDeliveryStreamsOutput: firehose.ListDeliveryStreamsOutput{
				DeliveryStreamNames: []string{"old", "new", "unreadable"},
			},
			CreationTimes: map[string]time.Time{
				"old": now.Add(-2 * time.Hour),
				"new": now,
			},
		}
	}

	mock := newMock()
	olderThanAnHour := config.ResourceType{ExcludeRule: config.FilterRule{TimeAfter: aws.Time(now.Add(-time.Hour))}}
	names, err := listKinesisFirehose(context.Background(), mock, resource.Scope{}, olderThanAnHour)
	require.NoError(t, err)
	require.Equal(t, []string{"old"}, aws.ToStringSlice(names), "an unreadable creation time leaves the stream out")

	mock = newMock()
	names, err = listKinesisFirehose(context.Background(), mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.Len(t, names, 3)
	require.Zero(t, mock.DescribeCalls, "no time filter, no DescribeDeliveryStream calls")
}

func TestDeleteKinesisFirehose(t *testing.T) {
	t.Parallel()

	mock := &mockKinesisFirehoseClient{}
	err := deleteKinesisFirehose(context.Background(), mock, aws.String("test-stream"))
	require.NoError(t, err)
}
