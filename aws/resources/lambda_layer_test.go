package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedLambdaLayer struct {
	lambdaiface.LambdaAPI
	ListLayersOutput         lambda.ListLayersOutput
	ListLayerVersionsOutput  lambda.ListLayerVersionsOutput
	DeleteLayerVersionOutput lambda.DeleteLayerVersionOutput
}

func (m mockedLambdaLayer) ListLayersPagesWithContext(_ aws.Context, input *lambda.ListLayersInput, fn func(*lambda.ListLayersOutput, bool) bool, _ ...request.Option) error {
	fn(&m.ListLayersOutput, true)
	return nil
}

func (m mockedLambdaLayer) ListLayerVersionsPagesWithContext(_ aws.Context, input *lambda.ListLayerVersionsInput, fn func(*lambda.ListLayerVersionsOutput, bool) bool, _ ...request.Option) error {
	fn(&m.ListLayerVersionsOutput, true)
	return nil
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

	ll := LambdaLayers{
		Client: mockedLambdaLayer{
			ListLayersOutput: lambda.ListLayersOutput{
				Layers: []*lambda.LayersListItem{
					{
						LayerName: aws.String(testName1),
						LatestMatchingVersion: &lambda.LayerVersionsListItem{
							CreatedDate: aws.String(testTimeStr),
						},
					},
					{
						LayerName: aws.String(testName2),
						LatestMatchingVersion: &lambda.LayerVersionsListItem{
							CreatedDate: aws.String(testTimeStr),
						},
					},
				},
			},
			ListLayerVersionsOutput: lambda.ListLayerVersionsOutput{
				LayerVersions: []*lambda.LayerVersionsListItem{
					{
						Version: &testName1Version1,
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
					TimeAfter: aws.Time(testTime.Add(-2 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := ll.getAll(context.Background(), config.Config{
				LambdaLayer: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestLambdaLayer_NukeAll(t *testing.T) {

	t.Parallel()

	ll := LambdaLayers{
		Client: mockedLambdaLayer{
			DeleteLayerVersionOutput: lambda.DeleteLayerVersionOutput{},
		},
	}

	err := ll.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
