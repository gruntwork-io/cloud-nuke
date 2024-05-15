package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedKinesisClient struct {
	kinesisiface.KinesisAPI
	ListStreamsOutput  kinesis.ListStreamsOutput
	DeleteStreamOutput kinesis.DeleteStreamOutput
}

func (m mockedKinesisClient) ListStreamsPagesWithContext(_ aws.Context, input *kinesis.ListStreamsInput, fn func(*kinesis.ListStreamsOutput, bool) bool, _ ...request.Option) error {
	fn(&m.ListStreamsOutput, true)
	return nil
}

func (m mockedKinesisClient) DeleteStreamWithContext(_ aws.Context, input *kinesis.DeleteStreamInput, _ ...request.Option) (*kinesis.DeleteStreamOutput, error) {
	return &m.DeleteStreamOutput, nil
}

func TestKinesisStreams_GetAll(t *testing.T) {

	t.Parallel()

	testName1 := "stream1"
	testName2 := "stream2"
	ks := KinesisStreams{
		Client: mockedKinesisClient{
			ListStreamsOutput: kinesis.ListStreamsOutput{
				StreamNames: []*string{aws.String(testName1), aws.String(testName2)},
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
				KinesisStream: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}

}

func TestKinesisStreams_NukeAll(t *testing.T) {

	t.Parallel()

	ks := KinesisStreams{
		Client: mockedKinesisClient{
			DeleteStreamOutput: kinesis.DeleteStreamOutput{},
		},
	}

	err := ks.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
