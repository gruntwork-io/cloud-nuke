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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockSageMakerEndpointConfigClient struct {
	ListEndpointConfigsOutput  sagemaker.ListEndpointConfigsOutput
	DeleteEndpointConfigOutput sagemaker.DeleteEndpointConfigOutput
	ListTagsOutput             sagemaker.ListTagsOutput
}

func (m *mockSageMakerEndpointConfigClient) ListEndpointConfigs(ctx context.Context, params *sagemaker.ListEndpointConfigsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListEndpointConfigsOutput, error) {
	return &m.ListEndpointConfigsOutput, nil
}

func (m *mockSageMakerEndpointConfigClient) DeleteEndpointConfig(ctx context.Context, params *sagemaker.DeleteEndpointConfigInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteEndpointConfigOutput, error) {
	return &m.DeleteEndpointConfigOutput, nil
}

func (m *mockSageMakerEndpointConfigClient) ListTags(ctx context.Context, params *sagemaker.ListTagsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListTagsOutput, error) {
	return &m.ListTagsOutput, nil
}

func TestListSageMakerEndpointConfigs(t *testing.T) {
	t.Parallel()

	testName1 := "endpoint-config-1"
	testName2 := "endpoint-config-2"
	now := time.Now()

	mock := &mockSageMakerEndpointConfigClient{
		ListEndpointConfigsOutput: sagemaker.ListEndpointConfigsOutput{
			EndpointConfigs: []types.EndpointConfigSummary{
				{EndpointConfigName: aws.String(testName1), CreationTime: aws.Time(now)},
				{EndpointConfigName: aws.String(testName2), CreationTime: aws.Time(now.Add(1 * time.Hour))},
			},
		},
		ListTagsOutput: sagemaker.ListTagsOutput{
			Tags: []types.Tag{
				{Key: aws.String("Environment"), Value: aws.String("test")},
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
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{testName1},
		},
		"tagIncludeFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"Environment": {RE: *regexp.MustCompile("test")},
					},
				},
			},
			expected: []string{testName1, testName2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), util.AccountIdKey, "123456789012")
			names, err := listSageMakerEndpointConfigs(ctx, mock, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteSageMakerEndpointConfig(t *testing.T) {
	t.Parallel()

	mock := &mockSageMakerEndpointConfigClient{}

	err := deleteSageMakerEndpointConfig(context.Background(), mock, aws.String("test-endpoint-config"))
	require.NoError(t, err)
}
