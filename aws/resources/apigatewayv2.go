package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// ApiGatewayV2API defines the interface for API Gateway V2 operations.
type ApiGatewayV2API interface {
	GetApis(ctx context.Context, params *apigatewayv2.GetApisInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetApisOutput, error)
	DeleteApi(ctx context.Context, params *apigatewayv2.DeleteApiInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.DeleteApiOutput, error)
}

// NewApiGatewayV2 creates a new ApiGatewayV2 resource using the generic resource pattern.
func NewApiGatewayV2() AwsResource {
	return NewAwsResource(&resource.Resource[ApiGatewayV2API]{
		ResourceTypeName: "apigatewayv2",
		BatchSize:        10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[ApiGatewayV2API], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = apigatewayv2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.APIGatewayV2
		},
		Lister: listApiGatewaysV2,
		Nuker:  resource.SimpleBatchDeleter(deleteApiGatewayV2),
	})
}

// listApiGatewaysV2 retrieves all API Gateways V2 that match the config filters.
// Note: apigatewayv2 SDK doesn't have built-in paginators, so we implement manual pagination.
func listApiGatewaysV2(ctx context.Context, client ApiGatewayV2API, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var ids []*string
	input := &apigatewayv2.GetApisInput{}

	for {
		output, err := client.GetApis(ctx, input)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, api := range output.Items {
			if cfg.ShouldInclude(config.ResourceValue{
				Time: api.CreatedDate,
				Name: api.Name,
				Tags: api.Tags,
			}) {
				ids = append(ids, api.ApiId)
			}
		}

		// Check for more pages
		if output.NextToken == nil {
			break
		}
		input.NextToken = output.NextToken
	}

	return ids, nil
}

// deleteApiGatewayV2 deletes a single API Gateway V2.
func deleteApiGatewayV2(ctx context.Context, client ApiGatewayV2API, apiID *string) error {
	_, err := client.DeleteApi(ctx, &apigatewayv2.DeleteApiInput{ApiId: apiID})
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
