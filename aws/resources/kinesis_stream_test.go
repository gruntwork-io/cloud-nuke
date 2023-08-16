package resources

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
)

type mockedKinesisClient struct {
	kinesisiface.KinesisAPI
	ListStreamsOutput  kinesis.ListStreamsOutput
	DeleteStreamOutput kinesis.DeleteStreamOutput
}

func (m mockedKinesisClient) ListStreamsPages(input *kinesis.ListStreamsInput, fn func(*kinesis.ListStreamsOutput, bool) bool) error {
	fn(&m.ListStreamsOutput, true)
	return nil
}

func (m mockedKinesisClient) DeleteStream(input *kinesis.DeleteStreamInput) (*kinesis.DeleteStreamOutput, error) {
	return &m.DeleteStreamOutput, nil
}

func TestKinesisStreams_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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
			names, err := ks.getAll(config.Config{
				KinesisStream: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}

}

func TestKinesisStreams_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	ks := KinesisStreams{
		Client: mockedKinesisClient{
			DeleteStreamOutput: kinesis.DeleteStreamOutput{},
		},
	}

	err := ks.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
