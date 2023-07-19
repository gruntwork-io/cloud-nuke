package aws

import (
	"regexp"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigatewayv2"
	"github.com/aws/aws-sdk-go/service/apigatewayv2/apigatewayv2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/assert"
)

type mockedApiGatewayV2 struct {
	apigatewayv2iface.ApiGatewayV2API
	GetApisOutput   apigatewayv2.GetApisOutput
	DeleteApiOutput apigatewayv2.DeleteApiOutput
}

func (m mockedApiGatewayV2) GetApis(*apigatewayv2.GetApisInput) (*apigatewayv2.GetApisOutput, error) {
	// Only need to return mocked response output
	return &m.GetApisOutput, nil
}

func (m mockedApiGatewayV2) DeleteApi(*apigatewayv2.DeleteApiInput) (*apigatewayv2.DeleteApiOutput, error) {
	// Only need to return mocked response output
	return &m.DeleteApiOutput, nil
}

func TestApiGatewayV2GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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
	apis, err := gw.getAll(config.Config{})
	assert.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(apis), testApiID)

	// filter by name
	apis, err = gw.getAll(config.Config{
		APIGatewayV2: config.ResourceType{
			ExcludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{{
					RE: *regexp.MustCompile("test-api-name"),
				}}}}})
	assert.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(apis), testApiID)

	// filter by date
	apis, err = gw.getAll(config.Config{
		APIGatewayV2: config.ResourceType{
			ExcludeRule: config.FilterRule{
				TimeAfter: awsgo.Time(now.Add(-1))}}})
	assert.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(apis), testApiID)
}

func TestApiGatewayV2NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	gw := ApiGatewayV2{
		Client: mockedApiGatewayV2{
			DeleteApiOutput: apigatewayv2.DeleteApiOutput{},
		},
	}
	err := gw.nukeAll([]*string{aws.String("test-api-id")})
	assert.NoError(t, err)
}
