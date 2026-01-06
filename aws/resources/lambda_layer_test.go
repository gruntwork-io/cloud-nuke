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

type mockLambdaLayerClient struct {
	ListLayersOutput         lambda.ListLayersOutput
	ListLayerVersionsOutput  lambda.ListLayerVersionsOutput
	DeleteLayerVersionOutput lambda.DeleteLayerVersionOutput
}

func (m *mockLambdaLayerClient) ListLayers(ctx context.Context, params *lambda.ListLayersInput, optFns ...func(*lambda.Options)) (*lambda.ListLayersOutput, error) {
	return &m.ListLayersOutput, nil
}

func (m *mockLambdaLayerClient) ListLayerVersions(ctx context.Context, params *lambda.ListLayerVersionsInput, optFns ...func(*lambda.Options)) (*lambda.ListLayerVersionsOutput, error) {
	return &m.ListLayerVersionsOutput, nil
}

func (m *mockLambdaLayerClient) DeleteLayerVersion(ctx context.Context, params *lambda.DeleteLayerVersionInput, optFns ...func(*lambda.Options)) (*lambda.DeleteLayerVersionOutput, error) {
	return &m.DeleteLayerVersionOutput, nil
}

func TestListLambdaLayers(t *testing.T) {
	t.Parallel()

	testName1 := "test-layer-1"
	testName2 := "test-layer-2"

	// Use fixed times that work with the AWS Lambda date format
	layout := "2006-01-02T15:04:05.000+0000"
	time1Str := "2023-07-28T10:00:00.000+0000"
	time2Str := "2023-07-28T12:00:00.000+0000"
	time1, _ := time.Parse(layout, time1Str)
	time2, _ := time.Parse(layout, time2Str)

	mock := &mockLambdaLayerClient{
		ListLayersOutput: lambda.ListLayersOutput{
			Layers: []types.LayersListItem{
				{
					LayerName: aws.String(testName1),
					LatestMatchingVersion: &types.LayerVersionsListItem{
						CreatedDate: aws.String(time1Str),
					},
				},
				{
					LayerName: aws.String(testName2),
					LatestMatchingVersion: &types.LayerVersionsListItem{
						CreatedDate: aws.String(time2Str),
					},
				},
			},
		},
		ListLayerVersionsOutput: lambda.ListLayerVersionsOutput{
			LayerVersions: []types.LayerVersionsListItem{
				{Version: 1},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testName1 + ":1", testName2 + ":1"},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testName2 + ":1"},
		},
		"timeAfterExclusionFilter": {
			// Exclude resources created after time1 + 1 hour (11:00)
			// time1 is 10:00, time2 is 12:00
			// So time1 should be included, time2 should be excluded
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(time1.Add(1 * time.Hour)),
				},
			},
			expected: []string{testName1 + ":1"},
		},
		"timeBeforeExclusionFilter": {
			// Exclude resources created before time2 (12:00)
			// So time1 should be excluded, time2 should be included
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeBefore: aws.Time(time2),
				},
			},
			expected: []string{testName2 + ":1"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listLambdaLayers(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteLambdaLayerVersion(t *testing.T) {
	t.Parallel()

	mock := &mockLambdaLayerClient{}

	// Valid deletion
	err := deleteLambdaLayerVersion(context.Background(), mock, aws.String("test-layer:1"))
	require.NoError(t, err)
}

func TestDeleteLambdaLayerVersion_InvalidIdentifier(t *testing.T) {
	t.Parallel()

	mock := &mockLambdaLayerClient{}

	// Missing colon separator
	err := deleteLambdaLayerVersion(context.Background(), mock, aws.String("invalid-identifier"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid layer version identifier")

	// Invalid version number
	err = deleteLambdaLayerVersion(context.Background(), mock, aws.String("test-layer:invalid"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid version number")
}
