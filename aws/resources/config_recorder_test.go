package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/aws/aws-sdk-go-v2/service/configservice/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedConfigServiceRecorders struct {
	ConfigServiceRecordersAPI
	DescribeConfigurationRecordersOutput configservice.DescribeConfigurationRecordersOutput
	DeleteConfigurationRecorderOutput    configservice.DeleteConfigurationRecorderOutput
}

func (m mockedConfigServiceRecorders) DescribeConfigurationRecorders(ctx context.Context, params *configservice.DescribeConfigurationRecordersInput, optFns ...func(*configservice.Options)) (*configservice.DescribeConfigurationRecordersOutput, error) {
	return &m.DescribeConfigurationRecordersOutput, nil
}

func (m mockedConfigServiceRecorders) DeleteConfigurationRecorder(ctx context.Context, params *configservice.DeleteConfigurationRecorderInput, optFns ...func(*configservice.Options)) (*configservice.DeleteConfigurationRecorderOutput, error) {
	return &m.DeleteConfigurationRecorderOutput, nil
}

func TestConfigServiceRecorder_GetAll(t *testing.T) {
	t.Parallel()

	testName1 := "test-recorder-1"
	testName2 := "test-recorder-2"
	csr := ConfigServiceRecorders{
		Client: mockedConfigServiceRecorders{
			DescribeConfigurationRecordersOutput: configservice.DescribeConfigurationRecordersOutput{
				ConfigurationRecorders: []types.ConfigurationRecorder{
					{Name: aws.String(testName1)},
					{Name: aws.String(testName2)},
				},
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
			names, err := csr.getAll(context.Background(), config.Config{
				ConfigServiceRecorder: tc.configObj,
			})

			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestConfigServiceRecorder_NukeAll(t *testing.T) {
	t.Parallel()

	csr := ConfigServiceRecorders{
		Client: mockedConfigServiceRecorders{
			DeleteConfigurationRecorderOutput: configservice.DeleteConfigurationRecorderOutput{},
		},
	}

	err := csr.nukeAll([]string{"test"})
	require.NoError(t, err)
}
