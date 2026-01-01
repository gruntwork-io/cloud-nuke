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
	"github.com/stretchr/testify/assert"
)

type mockedApiGatewayV2 struct {
	ApiGatewayV2API
	GetApisOutput          apigatewayv2.GetApisOutput
	GetDomainNamesOutput   apigatewayv2.GetDomainNamesOutput
	GetApiMappingsOutput   apigatewayv2.GetApiMappingsOutput
	DeleteApiOutput        apigatewayv2.DeleteApiOutput
	DeleteApiMappingOutput apigatewayv2.DeleteApiMappingOutput
}

func (m mockedApiGatewayV2) DeleteApi(ctx context.Context, params *apigatewayv2.DeleteApiInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.DeleteApiOutput, error) {
	return &m.DeleteApiOutput, nil
}

func (m mockedApiGatewayV2) DeleteApiMapping(ctx context.Context, params *apigatewayv2.DeleteApiMappingInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.DeleteApiMappingOutput, error) {
	return &m.DeleteApiMappingOutput, nil
}

func (m mockedApiGatewayV2) GetApis(ctx context.Context, params *apigatewayv2.GetApisInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetApisOutput, error) {
	return &m.GetApisOutput, nil
}

func (m mockedApiGatewayV2) GetApiMappings(ctx context.Context, params *apigatewayv2.GetApiMappingsInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetApiMappingsOutput, error) {
	return &m.GetApiMappingsOutput, nil
}

func (m mockedApiGatewayV2) GetDomainNames(ctx context.Context, params *apigatewayv2.GetDomainNamesInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetDomainNamesOutput, error) {
	return &m.GetDomainNamesOutput, nil
}

func TestApiGatewayV2GetAll(t *testing.T) {
	t.Parallel()

	testApiID := "test-api-id"
	testApiName := "test-api-name"
	now := time.Now()
	client := mockedApiGatewayV2{
		GetApisOutput: apigatewayv2.GetApisOutput{
			Items: []types.Api{
				{
					ApiId:       aws.String(testApiID),
					Name:        aws.String(testApiName),
					CreatedDate: aws.Time(now),
				},
			},
		},
	}

	// empty filter
	apis, err := listApiGatewaysV2(context.Background(), client, resource.Scope{Region: "us-east-1"}, config.ResourceType{})
	assert.NoError(t, err)
	assert.Contains(t, aws.ToStringSlice(apis), testApiID)

	// filter by name
	apis, err = listApiGatewaysV2(context.Background(), client, resource.Scope{Region: "us-east-1"}, config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{
				RE: *regexp.MustCompile("test-api-name"),
			}}}})
	assert.NoError(t, err)
	assert.NotContains(t, aws.ToStringSlice(apis), testApiID)

	// filter by date
	apis, err = listApiGatewaysV2(context.Background(), client, resource.Scope{Region: "us-east-1"}, config.ResourceType{
		ExcludeRule: config.FilterRule{
			TimeAfter: aws.Time(now.Add(-1))}})
	assert.NoError(t, err)
	assert.NotContains(t, aws.ToStringSlice(apis), testApiID)

	// filter by tags
	clientWithTags := mockedApiGatewayV2{
		GetApisOutput: apigatewayv2.GetApisOutput{
			Items: []types.Api{{
				ApiId:       aws.String(testApiID),
				Name:        aws.String(testApiName),
				CreatedDate: aws.Time(now),
				Tags:        map[string]string{"Environment": "production"},
			}},
		},
	}
	apis, err = listApiGatewaysV2(context.Background(), clientWithTags, resource.Scope{Region: "us-east-1"}, config.ResourceType{
		IncludeRule: config.FilterRule{
			Tags: map[string]config.Expression{
				"Environment": {RE: *regexp.MustCompile("production")},
			}}})
	assert.NoError(t, err)
	assert.Contains(t, aws.ToStringSlice(apis), testApiID)

	apis, err = listApiGatewaysV2(context.Background(), clientWithTags, resource.Scope{Region: "us-east-1"}, config.ResourceType{
		ExcludeRule: config.FilterRule{
			Tags: map[string]config.Expression{
				"Environment": {RE: *regexp.MustCompile("production")},
			}}})
	assert.NoError(t, err)
	assert.NotContains(t, aws.ToStringSlice(apis), testApiID)
}

func TestApiGatewayV2NukeAll(t *testing.T) {
	t.Parallel()

	client := mockedApiGatewayV2{
		DeleteApiOutput: apigatewayv2.DeleteApiOutput{},
		GetDomainNamesOutput: apigatewayv2.GetDomainNamesOutput{
			Items: []types.DomainName{
				{
					DomainName: aws.String("test-domain-name"),
				},
			},
		},
		GetApisOutput: apigatewayv2.GetApisOutput{
			Items: []types.Api{
				{
					ApiId: aws.String("test-api-id"),
				},
			},
		},
		DeleteApiMappingOutput: apigatewayv2.DeleteApiMappingOutput{},
	}
	err := deleteApiGatewaysV2(context.Background(), client, resource.Scope{Region: "us-east-1"}, "apigatewayv2", []*string{aws.String("test-api-id")})
	assert.NoError(t, err)
}
