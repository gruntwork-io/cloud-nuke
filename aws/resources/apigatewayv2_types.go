package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type ApiGatewayV2API interface {
	GetApis(ctx context.Context, params *apigatewayv2.GetApisInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetApisOutput, error)
	GetDomainNames(ctx context.Context, params *apigatewayv2.GetDomainNamesInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetDomainNamesOutput, error)
	GetApiMappings(ctx context.Context, params *apigatewayv2.GetApiMappingsInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetApiMappingsOutput, error)
	DeleteApi(ctx context.Context, params *apigatewayv2.DeleteApiInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.DeleteApiOutput, error)
	DeleteApiMapping(ctx context.Context, params *apigatewayv2.DeleteApiMappingInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.DeleteApiMappingOutput, error)
}

type ApiGatewayV2 struct {
	BaseAwsResource
	Client ApiGatewayV2API
	Region string
	Ids    []string
}

func (gw *ApiGatewayV2) InitV2(cfg aws.Config) {
	gw.Client = apigatewayv2.NewFromConfig(cfg)
}

func (gw *ApiGatewayV2) IsUsingV2() bool { return true }

func (gw *ApiGatewayV2) ResourceName() string {
	return "apigatewayv2"
}

func (gw *ApiGatewayV2) ResourceIdentifiers() []string {
	return gw.Ids
}

func (gw *ApiGatewayV2) MaxBatchSize() int {
	return 10
}

func (gw *ApiGatewayV2) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.APIGatewayV2
}

func (gw *ApiGatewayV2) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := gw.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	gw.Ids = aws.ToStringSlice(identifiers)
	return gw.Ids, nil
}

func (gw *ApiGatewayV2) Nuke(identifiers []string) error {
	if err := gw.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type TooManyApiGatewayV2Err struct{}

func (err TooManyApiGatewayV2Err) Error() string {
	return "Too many Api Gateways requested at once."
}
