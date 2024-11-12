package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedKinesisClient struct {
	KinesisStreamsAPI
	ListStreamsOutput  kinesis.ListStreamsOutput
	DeleteStreamOutput kinesis.DeleteStreamOutput
}

func (m mockedKinesisClient) ListStreams(ctx context.Context, params *kinesis.ListStreamsInput, optFns ...func(*kinesis.Options)) (*kinesis.ListStreamsOutput, error) {
	return &m.ListStreamsOutput, nil
}

func (m mockedKinesisClient) DeleteStream(ctx context.Context, params *kinesis.DeleteStreamInput, optFns ...func(*kinesis.Options)) (*kinesis.DeleteStreamOutput, error) {
	return &m.DeleteStreamOutput, nil
}

func TestKinesisStreams_GetAll(t *testing.T) {
	t.Parallel()

	testName1 := "stream1"
	testName2 := "stream2"
	ks := KinesisStreams{
		Client: mockedKinesisClient{
			ListStreamsOutput: kinesis.ListStreamsOutput{
				StreamNames: []string{testName1, testName2},
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
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
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
