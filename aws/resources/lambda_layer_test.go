package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockedLambdaLayer struct {
	LambdaLayersAPI
	DeleteLayerVersionOutput lambda.DeleteLayerVersionOutput
	ListLayersOutput         lambda.ListLayersOutput
	ListLayerVersionsOutput  lambda.ListLayerVersionsOutput
}

func (m mockedLambdaLayer) DeleteLayerVersion(ctx context.Context, params *lambda.DeleteLayerVersionInput, optFns ...func(*lambda.Options)) (*lambda.DeleteLayerVersionOutput, error) {
	return &m.DeleteLayerVersionOutput, nil
}

func (m mockedLambdaLayer) ListLayers(ctx context.Context, params *lambda.ListLayersInput, optFns ...func(*lambda.Options)) (*lambda.ListLayersOutput, error) {
	return &m.ListLayersOutput, nil
}

func (m mockedLambdaLayer) ListLayerVersions(ctx context.Context, params *lambda.ListLayerVersionsInput, optFns ...func(*lambda.Options)) (*lambda.ListLayerVersionsOutput, error) {
	return &m.ListLayerVersionsOutput, nil
}

func TestLambdaLayer_GetAll(t *testing.T) {
	t.Parallel()

	testName1 := "test-lambda-layer1"
	testName2 := "test-lambda-layer2"

	layout := "2006-01-02T15:04:05.000+0000"
	testTimeStr := "2023-07-28T12:34:56.789+0000"
	testTime, err := time.Parse(layout, testTimeStr)
	require.NoError(t, err)

	client := mockedLambdaLayer{
		ListLayersOutput: lambda.ListLayersOutput{
			Layers: []types.LayersListItem{
				{
					LayerName: aws.String(testName1),
					LatestMatchingVersion: &types.LayerVersionsListItem{
						CreatedDate: aws.String(testTimeStr),
					},
				},
				{
					LayerName: aws.String(testName2),
					LatestMatchingVersion: &types.LayerVersionsListItem{
						CreatedDate: aws.String(testTimeStr),
					},
				},
			},
		},
		ListLayerVersionsOutput: lambda.ListLayerVersionsOutput{
			LayerVersions: []types.LayerVersionsListItem{
				{
					Version: 1,
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
			// Each layer has one version (1), so identifiers are "layerName:1"
			expected: []string{testName1 + ":1", testName2 + ":1"},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testName2 + ":1"},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(testTime.Add(-2 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listLambdaLayers(context.Background(), client, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestLambdaLayer_NukeAll(t *testing.T) {
	t.Parallel()

	client := mockedLambdaLayer{
		DeleteLayerVersionOutput: lambda.DeleteLayerVersionOutput{},
	}

	// Test deleting a layer version with composite identifier
	err := deleteLambdaLayerVersion(context.Background(), client, aws.String("test-layer:1"))
	require.NoError(t, err)

	// Test with multiple versions
	err = deleteLambdaLayerVersion(context.Background(), client, aws.String("test-layer:2"))
	require.NoError(t, err)

	// Test with invalid identifier (no colon)
	err = deleteLambdaLayerVersion(context.Background(), client, aws.String("invalid-identifier"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid layer version identifier")

	// Test with invalid version number
	err = deleteLambdaLayerVersion(context.Background(), client, aws.String("test-layer:invalid"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid version number")
}
