package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/firehose/firehoseiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedKinesisFirehose struct {
	firehoseiface.FirehoseAPI
	ListDeliveryStreamsOutput  firehose.ListDeliveryStreamsOutput
	DeleteDeliveryStreamOutput firehose.DeleteDeliveryStreamOutput
}

func (m mockedKinesisFirehose) ListDeliveryStreamsWithContext(aws.Context, *firehose.ListDeliveryStreamsInput, ...request.Option) (*firehose.ListDeliveryStreamsOutput, error) {
	return &m.ListDeliveryStreamsOutput, nil
}

func (m mockedKinesisFirehose) DeleteDeliveryStreamWithContext(aws.Context, *firehose.DeleteDeliveryStreamInput, ...request.Option) (*firehose.DeleteDeliveryStreamOutput, error) {
	return &m.DeleteDeliveryStreamOutput, nil
}

func TestKinesisFirehoseStreams_GetAll(t *testing.T) {

	t.Parallel()

	testName1 := "stream1"
	testName2 := "stream2"
	ks := KinesisFirehose{
		Client: mockedKinesisFirehose{
			ListDeliveryStreamsOutput: firehose.ListDeliveryStreamsOutput{
				DeliveryStreamNames: []*string{aws.String(testName1), aws.String(testName2)},
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
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
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
