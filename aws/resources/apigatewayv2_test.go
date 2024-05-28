package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigatewayv2"
	"github.com/aws/aws-sdk-go/service/apigatewayv2/apigatewayv2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/assert"
)

type mockedApiGatewayV2 struct {
	apigatewayv2iface.ApiGatewayV2API
	GetApisOutput          apigatewayv2.GetApisOutput
	DeleteApiOutput        apigatewayv2.DeleteApiOutput
	GetDomainNamesOutput   apigatewayv2.GetDomainNamesOutput
	GetApiMappingsOutput   apigatewayv2.GetApiMappingsOutput
	DeleteApiMappingOutput apigatewayv2.DeleteApiMappingOutput
}

func (m mockedApiGatewayV2) GetApisWithContext(_ awsgo.Context, _ *apigatewayv2.GetApisInput, _ ...request.Option) (*apigatewayv2.GetApisOutput, error) {
	// Only need to return mocked response output
	return &m.GetApisOutput, nil
}

func (m mockedApiGatewayV2) DeleteApiWithContext(_ awsgo.Context, _ *apigatewayv2.DeleteApiInput, _ ...request.Option) (*apigatewayv2.DeleteApiOutput, error) {
	// Only need to return mocked response output
	return &m.DeleteApiOutput, nil
}

func (m mockedApiGatewayV2) GetDomainNames(_ *apigatewayv2.GetDomainNamesInput) (*apigatewayv2.GetDomainNamesOutput, error) {
	return &m.GetDomainNamesOutput, nil
}

func (m mockedApiGatewayV2) GetApiMappings(_ *apigatewayv2.GetApiMappingsInput) (*apigatewayv2.GetApiMappingsOutput, error) {
	return &m.GetApiMappingsOutput, nil
}

func (m mockedApiGatewayV2) DeleteApiMappingWithContext(_ awsgo.Context, _ *apigatewayv2.DeleteApiMappingInput, _ ...request.Option) (*apigatewayv2.DeleteApiMappingOutput, error) {
	return &m.DeleteApiMappingOutput, nil
}

func TestApiGatewayV2GetAll(t *testing.T) {

	t.Parallel()

	testApiID := "test-api-id"
	testApiName := "test-api-name"
	now := time.Now()
	gw := ApiGatewayV2{
		Client: mockedApiGatewayV2{
			GetApisOutput: apigatewayv2.GetApisOutput{
				Items: []*apigatewayv2.Api{
					{
						ApiId:       aws.String(testApiID),
						Name:        aws.String(testApiName),
						CreatedDate: aws.Time(now),
					},
				},
			},
		},
	}

	// empty filter
	apis, err := gw.getAll(context.Background(), config.Config{})
	assert.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(apis), testApiID)

	// filter by name
	apis, err = gw.getAll(context.Background(), config.Config{
		APIGatewayV2: config.ResourceType{
			ExcludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{{
					RE: *regexp.MustCompile("test-api-name"),
				}}}}})
	assert.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(apis), testApiID)

	// filter by date
	apis, err = gw.getAll(context.Background(), config.Config{
		APIGatewayV2: config.ResourceType{
			ExcludeRule: config.FilterRule{
				TimeAfter: awsgo.Time(now.Add(-1))}}})
	assert.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(apis), testApiID)
}

func TestApiGatewayV2NukeAll(t *testing.T) {

	t.Parallel()

	gw := ApiGatewayV2{
		Client: mockedApiGatewayV2{
			DeleteApiOutput: apigatewayv2.DeleteApiOutput{},
			GetDomainNamesOutput: apigatewayv2.GetDomainNamesOutput{
				Items: []*apigatewayv2.DomainName{
					{
						DomainName: aws.String("test-domain-name"),
					},
				},
			},
			GetApisOutput: apigatewayv2.GetApisOutput{
				Items: []*apigatewayv2.Api{
					{
						ApiId: aws.String("test-api-id"),
					},
				},
			},
			DeleteApiMappingOutput: apigatewayv2.DeleteApiMappingOutput{},
		},
	}
	err := gw.nukeAll([]*string{aws.String("test-api-id")})
	assert.NoError(t, err)
}
