package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type ApiGatewayServiceAPI interface {
	GetRestApis(ctx context.Context, params *apigateway.GetRestApisInput, optFns ...func(*apigateway.Options)) (*apigateway.GetRestApisOutput, error)
	GetStages(ctx context.Context, params *apigateway.GetStagesInput, optFns ...func(*apigateway.Options)) (*apigateway.GetStagesOutput, error)
	DeleteClientCertificate(ctx context.Context, params *apigateway.DeleteClientCertificateInput, optFns ...func(*apigateway.Options)) (*apigateway.DeleteClientCertificateOutput, error)
	DeleteRestApi(ctx context.Context, params *apigateway.DeleteRestApiInput, optFns ...func(*apigateway.Options)) (*apigateway.DeleteRestApiOutput, error)
}

type ApiGateway struct {
	BaseAwsResource
	Client ApiGatewayServiceAPI
	Region string
	Ids    []string
}

func (gateway *ApiGateway) InitV2(cfg aws.Config) {
	gateway.Client = apigateway.NewFromConfig(cfg)
}

func (gateway *ApiGateway) IsUsingV2() bool { return true }

func (gateway *ApiGateway) ResourceName() string {
	return "apigateway"
}

func (gateway *ApiGateway) ResourceIdentifiers() []string {
	return gateway.Ids
}

func (gateway *ApiGateway) MaxBatchSize() int {
	return 10
}

func (gateway *ApiGateway) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.APIGateway
}

func (gateway *ApiGateway) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := gateway.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	gateway.Ids = aws.ToStringSlice(identifiers)
	return gateway.Ids, nil
}

func (gateway *ApiGateway) Nuke(identifiers []string) error {
	if err := gateway.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type TooManyApiGatewayErr struct{}

func (err TooManyApiGatewayErr) Error() string {
	return "Too many Api Gateways requested at once."
}
