package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// ApiGatewayAPI defines the interface for API Gateway (v1) operations.
type ApiGatewayAPI interface {
	GetRestApis(ctx context.Context, params *apigateway.GetRestApisInput, optFns ...func(*apigateway.Options)) (*apigateway.GetRestApisOutput, error)
	DeleteRestApi(ctx context.Context, params *apigateway.DeleteRestApiInput, optFns ...func(*apigateway.Options)) (*apigateway.DeleteRestApiOutput, error)
	GetDomainNames(ctx context.Context, params *apigateway.GetDomainNamesInput, optFns ...func(*apigateway.Options)) (*apigateway.GetDomainNamesOutput, error)
	GetBasePathMappings(ctx context.Context, params *apigateway.GetBasePathMappingsInput, optFns ...func(*apigateway.Options)) (*apigateway.GetBasePathMappingsOutput, error)
	DeleteBasePathMapping(ctx context.Context, params *apigateway.DeleteBasePathMappingInput, optFns ...func(*apigateway.Options)) (*apigateway.DeleteBasePathMappingOutput, error)
}

// NewApiGateway creates a new ApiGateway resource using the generic resource pattern.
func NewApiGateway() AwsResource {
	return NewAwsResource(&resource.Resource[ApiGatewayAPI]{
		ResourceTypeName: "api-gateway",
		BatchSize:        10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[ApiGatewayAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = apigateway.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.APIGateway
		},
		Lister: listApiGateways,
		// AWS refuses to delete a REST API that a custom domain name still maps to, so the mappings go first.
		Nuker: nukeApiGateways,
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

// apiGatewayBasePathMapping is one custom domain name base path mapping, found by listApiGatewayBasePathMappings.
type apiGatewayBasePathMapping struct {
	domainName *string
	// domainNameID is set only for a private custom domain name, which AWS requires it for.
	domainNameID *string
	basePath     *string
}

// nukeApiGateways deletes each REST API after the custom domain name base path mappings that point at it. The
// mappings are listed once per batch rather than once per API, since every API would otherwise scan every domain.
func nukeApiGateways(ctx context.Context, client ApiGatewayAPI, scope resource.Scope, resourceType string, ids []*string) []resource.NukeResult {
	mappings, err := listApiGatewayBasePathMappings(ctx, client)
	if err != nil {
		// Still try each delete: an API with no mapping deletes, and AWS names any mapping that blocks one.
		logging.Warnf("[API Gateway] Unable to list custom domain name base path mappings, so none are removed first: %v", err)
	}

	return resource.SimpleBatchDeleter(func(ctx context.Context, client ApiGatewayAPI, apiID *string) error {
		if err := deleteApiGatewayBasePathMappings(ctx, client, mappings[aws.ToString(apiID)]); err != nil {
			return err
		}
		return deleteApiGateway(ctx, client, apiID)
	})(ctx, client, scope, resourceType, ids)
}

// listApiGatewayBasePathMappings returns every custom domain name base path mapping in the region, keyed by the
// REST API it points at.
func listApiGatewayBasePathMappings(ctx context.Context, client ApiGatewayAPI) (map[string][]apiGatewayBasePathMapping, error) {
	mappings := map[string][]apiGatewayBasePathMapping{}

	domains := apigateway.NewGetDomainNamesPaginator(client, &apigateway.GetDomainNamesInput{})
	for domains.HasMorePages() {
		page, err := domains.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list domain names: %w", err)
		}

		for _, domain := range page.Items {
			pages := apigateway.NewGetBasePathMappingsPaginator(client, &apigateway.GetBasePathMappingsInput{
				DomainName:   domain.DomainName,
				DomainNameId: domain.DomainNameId,
			})
			for pages.HasMorePages() {
				mappingPage, err := pages.NextPage(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to list base path mappings on domain %s: %w", aws.ToString(domain.DomainName), err)
				}

				for _, mapping := range mappingPage.Items {
					apiID := aws.ToString(mapping.RestApiId)
					mappings[apiID] = append(mappings[apiID], apiGatewayBasePathMapping{
						domainName:   domain.DomainName,
						domainNameID: domain.DomainNameId,
						basePath:     mapping.BasePath,
					})
				}
			}
		}
	}

	return mappings, nil
}

// deleteApiGatewayBasePathMappings deletes the given custom domain name base path mappings of one REST API.
func deleteApiGatewayBasePathMappings(ctx context.Context, client ApiGatewayAPI, mappings []apiGatewayBasePathMapping) error {
	for _, mapping := range mappings {
		logging.Debugf("[API Gateway] Deleting base path mapping %s on domain %s",
			aws.ToString(mapping.basePath), aws.ToString(mapping.domainName))
		_, err := client.DeleteBasePathMapping(ctx, &apigateway.DeleteBasePathMappingInput{
			BasePath:     mapping.basePath,
			DomainName:   mapping.domainName,
			DomainNameId: mapping.domainNameID,
		})
		if err != nil {
			return fmt.Errorf("failed to delete base path mapping %s on domain %s: %w",
				aws.ToString(mapping.basePath), aws.ToString(mapping.domainName), err)
		}
	}
	return nil
}

// deleteApiGateway deletes a single API Gateway REST API.
func deleteApiGateway(ctx context.Context, client ApiGatewayAPI, apiID *string) error {
	_, err := client.DeleteRestApi(ctx, &apigateway.DeleteRestApiInput{RestApiId: apiID})
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
