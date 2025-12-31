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
	testName1Version1 := int64(1)

	testName2 := "test-lambda-layer2"

	testTime := time.Now()

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
					Version: testName1Version1,
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
		ListLayerVersionsOutput:  lambda.ListLayerVersionsOutput{},
	}

	err := deleteLambdaLayers(context.Background(), client, resource.Scope{Region: "us-east-1"}, "lambda_layer", []*string{aws.String("test")})
	require.NoError(t, err)
}
