package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockedApiGatewayV2 struct {
	ApiGatewayV2API
	GetApisOutput   apigatewayv2.GetApisOutput
	DeleteApiOutput apigatewayv2.DeleteApiOutput
}

func (m mockedApiGatewayV2) GetApis(ctx context.Context, params *apigatewayv2.GetApisInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetApisOutput, error) {
	return &m.GetApisOutput, nil
}

func (m mockedApiGatewayV2) DeleteApi(ctx context.Context, params *apigatewayv2.DeleteApiInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.DeleteApiOutput, error) {
	return &m.DeleteApiOutput, nil
}

func TestApiGatewayV2_GetAll(t *testing.T) {
	t.Parallel()

	testApiID1 := "api-1"
	testApiID2 := "api-2"
	testApiName1 := "test-api-1"
	testApiName2 := "test-api-2"
	now := time.Now()

	mock := mockedApiGatewayV2{
		GetApisOutput: apigatewayv2.GetApisOutput{
			Items: []types.Api{
				{
					ApiId:       aws.String(testApiID1),
					Name:        aws.String(testApiName1),
					CreatedDate: aws.Time(now),
					Tags:        map[string]string{"env": "dev"},
				},
				{
					ApiId:       aws.String(testApiID2),
					Name:        aws.String(testApiName2),
					CreatedDate: aws.Time(now.Add(1 * time.Hour)),
					Tags:        map[string]string{"env": "prod"},
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
			expected:  []string{testApiID1, testApiID2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile("test-api-1"),
					}},
				},
			},
			expected: []string{testApiID2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{testApiID1},
		},
		"tagExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"env": {RE: *regexp.MustCompile("prod")},
					},
				},
			},
			expected: []string{testApiID1},
		},
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"env": {RE: *regexp.MustCompile("prod")},
					},
				},
			},
			expected: []string{testApiID2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			apis, err := listApiGatewaysV2(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(apis))
		})
	}
}

func TestApiGatewayV2_NukeAll(t *testing.T) {
	t.Parallel()

	mock := mockedApiGatewayV2{
		DeleteApiOutput: apigatewayv2.DeleteApiOutput{},
	}

	err := deleteApiGatewayV2(context.Background(), mock, aws.String("api-1"))
	require.NoError(t, err)
}
