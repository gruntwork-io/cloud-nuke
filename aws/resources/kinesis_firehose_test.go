package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedKinesisFirehose struct {
	KinesisFirehoseAPI
	DeleteDeliveryStreamOutput firehose.DeleteDeliveryStreamOutput
	ListDeliveryStreamsOutput  firehose.ListDeliveryStreamsOutput
}

func (m mockedKinesisFirehose) DeleteDeliveryStream(ctx context.Context, params *firehose.DeleteDeliveryStreamInput, optFns ...func(*firehose.Options)) (*firehose.DeleteDeliveryStreamOutput, error) {
	return &m.DeleteDeliveryStreamOutput, nil
}

func (m mockedKinesisFirehose) ListDeliveryStreams(ctx context.Context, params *firehose.ListDeliveryStreamsInput, optFns ...func(*firehose.Options)) (*firehose.ListDeliveryStreamsOutput, error) {
	return &m.ListDeliveryStreamsOutput, nil
}

func TestKinesisFirehoseStreams_GetAll(t *testing.T) {
	t.Parallel()
	testName1 := "stream1"
	testName2 := "stream2"
	ks := KinesisFirehose{
		Client: mockedKinesisFirehose{
			ListDeliveryStreamsOutput: firehose.ListDeliveryStreamsOutput{
				DeliveryStreamNames: []string{testName1, testName2},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testName2},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := ks.getAll(context.Background(), config.Config{
				KinesisFirehose: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}

}

func TestKinesisFirehose_NukeAll(t *testing.T) {
	t.Parallel()
	ks := KinesisFirehose{
		Client: mockedKinesisFirehose{
			DeleteDeliveryStreamOutput: firehose.DeleteDeliveryStreamOutput{},
		},
	}

	err := ks.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
