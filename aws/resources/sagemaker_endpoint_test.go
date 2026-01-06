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

type mockSageMakerEndpointClient struct {
	ListEndpointsOutput  sagemaker.ListEndpointsOutput
	DeleteEndpointOutput sagemaker.DeleteEndpointOutput
	ListTagsOutput       sagemaker.ListTagsOutput
}

func (m *mockSageMakerEndpointClient) ListEndpoints(ctx context.Context, params *sagemaker.ListEndpointsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListEndpointsOutput, error) {
	return &m.ListEndpointsOutput, nil
}

func (m *mockSageMakerEndpointClient) DeleteEndpoint(ctx context.Context, params *sagemaker.DeleteEndpointInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteEndpointOutput, error) {
	return &m.DeleteEndpointOutput, nil
}

func (m *mockSageMakerEndpointClient) ListTags(ctx context.Context, params *sagemaker.ListTagsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListTagsOutput, error) {
	return &m.ListTagsOutput, nil
}

func TestListSageMakerEndpoints(t *testing.T) {
	t.Parallel()

	testName1 := "endpoint-1"
	testName2 := "endpoint-2"
	now := time.Now()

	mock := &mockSageMakerEndpointClient{
		ListEndpointsOutput: sagemaker.ListEndpointsOutput{
			Endpoints: []types.EndpointSummary{
				{
					EndpointName:   aws.String(testName1),
					CreationTime:   aws.Time(now),
					EndpointStatus: types.EndpointStatusInService,
				},
				{
					EndpointName:   aws.String(testName2),
					CreationTime:   aws.Time(now.Add(1 * time.Hour)),
					EndpointStatus: types.EndpointStatusInService,
				},
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
		"tagInclusionFilter": {
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
			ctx := context.WithValue(context.Background(), util.AccountIdKey, "test-account-id")
			names, err := listSageMakerEndpoints(ctx, mock, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteSageMakerEndpoint(t *testing.T) {
	t.Parallel()

	mock := &mockSageMakerEndpointClient{
		DeleteEndpointOutput: sagemaker.DeleteEndpointOutput{},
	}

	err := deleteSageMakerEndpoint(context.Background(), mock, aws.String("test-endpoint"))
	require.NoError(t, err)
}
