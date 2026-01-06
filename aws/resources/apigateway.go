package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// ApiGatewayAPI defines the interface for API Gateway (v1) operations.
type ApiGatewayAPI interface {
	GetRestApis(ctx context.Context, params *apigateway.GetRestApisInput, optFns ...func(*apigateway.Options)) (*apigateway.GetRestApisOutput, error)
	DeleteRestApi(ctx context.Context, params *apigateway.DeleteRestApiInput, optFns ...func(*apigateway.Options)) (*apigateway.DeleteRestApiOutput, error)
}

// NewApiGateway creates a new ApiGateway resource using the generic resource pattern.
func NewApiGateway() AwsResource {
	return NewAwsResource(&resource.Resource[ApiGatewayAPI]{
		ResourceTypeName: "apigateway",
		BatchSize:        10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[ApiGatewayAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = apigateway.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.APIGateway
		},
		Lister: listApiGateways,
		Nuker:  resource.SimpleBatchDeleter(deleteApiGateway),
	})
}

// listApiGateways retrieves all API Gateway (v1) REST APIs that match the config filters.
func listApiGateways(ctx context.Context, client ApiGatewayAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var ids []*string

	paginator := apigateway.NewGetRestApisPaginator(client, &apigateway.GetRestApisInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, api := range page.Items {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: api.Name,
				Time: api.CreatedDate,
				Tags: api.Tags,
			}) {
				ids = append(ids, api.Id)
			}
		}
	}

	return ids, nil
}

// deleteApiGateway deletes a single API Gateway REST API.
func deleteApiGateway(ctx context.Context, client ApiGatewayAPI, apiID *string) error {
	_, err := client.DeleteRestApi(ctx, &apigateway.DeleteRestApiInput{RestApiId: apiID})
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
