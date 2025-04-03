package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedSageMakerEndpointConfig struct {
	SageMakerEndpointConfigAPI
	ListEndpointConfigsOutput  sagemaker.ListEndpointConfigsOutput
	DeleteEndpointConfigOutput sagemaker.DeleteEndpointConfigOutput
	ListTagsOutput             sagemaker.ListTagsOutput
}

func (m mockedSageMakerEndpointConfig) ListEndpointConfigs(ctx context.Context, params *sagemaker.ListEndpointConfigsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListEndpointConfigsOutput, error) {
	return &m.ListEndpointConfigsOutput, nil
}

func (m mockedSageMakerEndpointConfig) DeleteEndpointConfig(ctx context.Context, params *sagemaker.DeleteEndpointConfigInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteEndpointConfigOutput, error) {
	return &m.DeleteEndpointConfigOutput, nil
}

func (m mockedSageMakerEndpointConfig) ListTags(ctx context.Context, params *sagemaker.ListTagsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListTagsOutput, error) {
	return &m.ListTagsOutput, nil
}

func TestSageMakerEndpointConfig_GetAll(t *testing.T) {
	t.Parallel()

	testName1 := "endpoint-config-1"
	testName2 := "endpoint-config-2"
	now := time.Now()

	endpointConfig := SageMakerEndpointConfig{
		Client: mockedSageMakerEndpointConfig{
			ListEndpointConfigsOutput: sagemaker.ListEndpointConfigsOutput{
				EndpointConfigs: []types.EndpointConfigSummary{
					{
						EndpointConfigName: aws.String(testName1),
						CreationTime:       aws.Time(now),
					},
					{
						EndpointConfigName: aws.String(testName2),
						CreationTime:       aws.Time(now.Add(1)),
					},
				},
			},
			ListTagsOutput: sagemaker.ListTagsOutput{
				Tags: []types.Tag{
					{
						Key:   aws.String("Environment"),
						Value: aws.String("test"),
					},
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
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testName1},
		},
		"tagFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"Environment": {RE: *regexp.MustCompile("test")},
					},
				}},
			expected: []string{testName1, testName2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			ctx = context.WithValue(ctx, util.AccountIdKey, "test-account-id")

			endpointConfigs, err := endpointConfig.GetAll(ctx, config.Config{
				SageMakerEndpointConfig: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(endpointConfigs))
		})
	}
}

func TestSageMakerEndpointConfig_NukeAll(t *testing.T) {
	t.Parallel()

	endpointConfig := SageMakerEndpointConfig{
		Client: mockedSageMakerEndpointConfig{
			DeleteEndpointConfigOutput: sagemaker.DeleteEndpointConfigOutput{},
		},
	}

	err := endpointConfig.nukeAll([]string{"endpoint-config-1", "endpoint-config-2"})
	require.NoError(t, err)
}

func TestSageMakerEndpointConfig_EmptyNukeAll(t *testing.T) {
	t.Parallel()

	endpointConfig := SageMakerEndpointConfig{
		Client: mockedSageMakerEndpointConfig{
			DeleteEndpointConfigOutput: sagemaker.DeleteEndpointConfigOutput{},
		},
	}

	err := endpointConfig.nukeAll([]string{})
	require.NoError(t, err)
}
