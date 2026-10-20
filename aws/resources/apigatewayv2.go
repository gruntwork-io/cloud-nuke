package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// ApiGatewayV2API defines the interface for API Gateway V2 operations.
type ApiGatewayV2API interface {
	GetApis(ctx context.Context, params *apigatewayv2.GetApisInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetApisOutput, error)
	DeleteApi(ctx context.Context, params *apigatewayv2.DeleteApiInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.DeleteApiOutput, error)
	GetDomainNames(ctx context.Context, params *apigatewayv2.GetDomainNamesInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetDomainNamesOutput, error)
	GetApiMappings(ctx context.Context, params *apigatewayv2.GetApiMappingsInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetApiMappingsOutput, error)
	DeleteApiMapping(ctx context.Context, params *apigatewayv2.DeleteApiMappingInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.DeleteApiMappingOutput, error)
}

// NewApiGatewayV2 creates a new ApiGatewayV2 resource using the generic resource pattern.
func NewApiGatewayV2() AwsResource {
	return NewAwsResource(&resource.Resource[ApiGatewayV2API]{
		ResourceTypeName: "api-gateway-v2",
		BatchSize:        10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[ApiGatewayV2API], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = apigatewayv2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.APIGatewayV2
		},
		Lister: listApiGatewaysV2,
		// AWS refuses to delete an API that a custom domain name still maps to, so the mappings go first.
		Nuker: nukeApiGatewaysV2,
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

// apiGatewayV2Mapping is one custom domain name mapping, found by listApiGatewayV2Mappings.
type apiGatewayV2Mapping struct {
	domainName *string
	mappingID  *string
}

// nukeApiGatewaysV2 deletes each API after the custom domain name mappings that point at it. The mappings are
// listed once per batch rather than once per API, since every API would otherwise scan every domain.
func nukeApiGatewaysV2(ctx context.Context, client ApiGatewayV2API, scope resource.Scope, resourceType string, ids []*string) []resource.NukeResult {
	mappings, err := listApiGatewayV2Mappings(ctx, client)
	if err != nil {
		// Still try each delete: an API with no mapping deletes, and AWS names any mapping that blocks one.
		logging.Warnf("[API Gateway V2] Unable to list every custom domain name mapping, so only those found are removed first: %v", err)
	}

	return resource.SimpleBatchDeleter(func(ctx context.Context, client ApiGatewayV2API, apiID *string) error {
		if err := deleteApiGatewayV2Mappings(ctx, client, mappings[aws.ToString(apiID)]); err != nil {
			return err
		}
		return deleteApiGatewayV2(ctx, client, apiID)
	})(ctx, client, scope, resourceType, ids)
}

// listApiGatewayV2Mappings returns every custom domain name mapping in the region, keyed by the API it points at.
// A domain whose mappings cannot be listed is skipped, so the mappings found on the others are still returned,
// together with the errors.
func listApiGatewayV2Mappings(ctx context.Context, client ApiGatewayV2API) (map[string][]apiGatewayV2Mapping, error) {
	mappings := map[string][]apiGatewayV2Mapping{}
	var errs *multierror.Error
	domainInput := &apigatewayv2.GetDomainNamesInput{}

	for {
		domains, err := client.GetDomainNames(ctx, domainInput)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("failed to list domain names: %w", err))
			return mappings, errs.ErrorOrNil()
		}

		for _, domain := range domains.Items {
			mappingInput := &apigatewayv2.GetApiMappingsInput{DomainName: domain.DomainName}
			for {
				page, err := client.GetApiMappings(ctx, mappingInput)
				if err != nil {
					errs = multierror.Append(errs, fmt.Errorf("failed to list API mappings on domain %s: %w", aws.ToString(domain.DomainName), err))
					break
				}

				for _, mapping := range page.Items {
					apiID := aws.ToString(mapping.ApiId)
					mappings[apiID] = append(mappings[apiID], apiGatewayV2Mapping{
						domainName: domain.DomainName,
						mappingID:  mapping.ApiMappingId,
					})
				}

				if page.NextToken == nil {
					break
				}
				mappingInput.NextToken = page.NextToken
			}
		}

		if domains.NextToken == nil {
			return mappings, errs.ErrorOrNil()
		}
		domainInput.NextToken = domains.NextToken
	}
}

// deleteApiGatewayV2Mappings deletes the given custom domain name mappings of one API.
func deleteApiGatewayV2Mappings(ctx context.Context, client ApiGatewayV2API, mappings []apiGatewayV2Mapping) error {
	for _, mapping := range mappings {
		logging.Debugf("[API Gateway V2] Deleting API mapping %s on domain %s",
			aws.ToString(mapping.mappingID), aws.ToString(mapping.domainName))
		_, err := client.DeleteApiMapping(ctx, &apigatewayv2.DeleteApiMappingInput{
			ApiMappingId: mapping.mappingID,
			DomainName:   mapping.domainName,
		})
		if err != nil {
			return fmt.Errorf("failed to delete API mapping %s on domain %s: %w",
				aws.ToString(mapping.mappingID), aws.ToString(mapping.domainName), err)
		}
	}
	return nil
}

// deleteApiGatewayV2 deletes a single API Gateway V2.
func deleteApiGatewayV2(ctx context.Context, client ApiGatewayV2API, apiID *string) error {
	_, err := client.DeleteApi(ctx, &apigatewayv2.DeleteApiInput{ApiId: apiID})
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
