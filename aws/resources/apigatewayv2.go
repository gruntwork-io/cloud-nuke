package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// ApiGatewayV2API defines the interface for API Gateway V2 operations.
type ApiGatewayV2API interface {
	GetApis(ctx context.Context, params *apigatewayv2.GetApisInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetApisOutput, error)
	GetDomainNames(ctx context.Context, params *apigatewayv2.GetDomainNamesInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetDomainNamesOutput, error)
	GetApiMappings(ctx context.Context, params *apigatewayv2.GetApiMappingsInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetApiMappingsOutput, error)
	DeleteApi(ctx context.Context, params *apigatewayv2.DeleteApiInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.DeleteApiOutput, error)
	DeleteApiMapping(ctx context.Context, params *apigatewayv2.DeleteApiMappingInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.DeleteApiMappingOutput, error)
}

// NewApiGatewayV2 creates a new ApiGatewayV2 resource using the generic resource pattern.
func NewApiGatewayV2() AwsResource {
	return NewAwsResource(&resource.Resource[ApiGatewayV2API]{
		ResourceTypeName: "apigatewayv2",
		BatchSize:        10,
		InitClient: func(r *resource.Resource[ApiGatewayV2API], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for ApiGatewayV2 client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = apigatewayv2.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.APIGatewayV2
		},
		Lister: listApiGatewaysV2,
		Nuker:  deleteApiGatewaysV2,
	})
}

// listApiGatewaysV2 retrieves all API Gateways V2 that match the config filters.
func listApiGatewaysV2(ctx context.Context, client ApiGatewayV2API, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	output, err := client.GetApis(ctx, &apigatewayv2.GetApisInput{})
	if err != nil {
		return []*string{}, errors.WithStackTrace(err)
	}

	var ids []*string
	for _, restapi := range output.Items {
		if cfg.ShouldInclude(config.ResourceValue{
			Time: restapi.CreatedDate,
			Name: restapi.Name,
			Tags: restapi.Tags,
		}) {
			ids = append(ids, restapi.ApiId)
		}
	}

	return ids, nil
}

// deleteApiGatewaysV2 is a custom nuker for API Gateway V2 resources.
// It first deletes associated API mappings, then deletes the APIs themselves.
func deleteApiGatewaysV2(ctx context.Context, client ApiGatewayV2API, scope resource.Scope, resourceType string, identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No API Gateways (v2) to nuke in %s", scope)
		return nil
	}

	if len(identifiers) > 100 {
		logging.Debugf("Nuking too many API Gateways (v2) at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyApiGatewayV2Err{}
	}

	err := deleteAssociatedApiMappings(ctx, client, identifiers)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	// There is no bulk delete Api Gateway API, so we delete the batch of gateways concurrently using goroutines
	logging.Debugf("Deleting Api Gateways (v2) in %s", scope)
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, apigwID := range identifiers {
		errChans[i] = make(chan error, 1)
		go deleteApiGatewayV2Async(ctx, client, scope, resourceType, wg, errChans[i], apigwID)
	}
	wg.Wait()

	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}
	return nil
}

func deleteApiGatewayV2Async(ctx context.Context, client ApiGatewayV2API, scope resource.Scope, resourceType string, wg *sync.WaitGroup, errChan chan error, apiId *string) {
	defer wg.Done()

	input := &apigatewayv2.DeleteApiInput{ApiId: apiId}
	_, err := client.DeleteApi(ctx, input)
	errChan <- err

	// Record status of this resource
	e := report.Entry{
		Identifier:   *apiId,
		ResourceType: resourceType,
		Error:        err,
	}
	report.Record(e)

	if err == nil {
		logging.Debugf("Successfully deleted API Gateway (v2) %s in %s", aws.ToString(apiId), scope)
	} else {
		logging.Debugf("Failed to delete API Gateway (v2) %s in %s", aws.ToString(apiId), scope)
	}
}

func deleteAssociatedApiMappings(ctx context.Context, client ApiGatewayV2API, identifiers []*string) error {
	// Convert identifiers to map to check if identifier is in list
	identifierMap := make(map[string]bool)
	for _, identifier := range identifiers {
		identifierMap[*identifier] = true
	}

	domainNames, err := client.GetDomainNames(ctx, &apigatewayv2.GetDomainNamesInput{})
	if err != nil {
		logging.Debugf("Failed to get domain names: %s", err)
		return errors.WithStackTrace(err)
	}

	logging.Debugf("Found %d domain names", len(domainNames.Items))
	for _, domainName := range domainNames.Items {
		apiMappings, err := client.GetApiMappings(ctx, &apigatewayv2.GetApiMappingsInput{
			DomainName: domainName.DomainName,
		})
		if err != nil {
			logging.Debugf("Failed to get api mappings: %s", err)
			return errors.WithStackTrace(err)
		}

		for _, apiMapping := range apiMappings.Items {
			if _, ok := identifierMap[*apiMapping.ApiId]; !ok {
				continue
			}

			_, err := client.DeleteApiMapping(ctx, &apigatewayv2.DeleteApiMappingInput{
				ApiMappingId: apiMapping.ApiMappingId,
				DomainName:   domainName.DomainName,
			})
			if err != nil {
				logging.Debugf("Failed to delete api mapping: %s", err)
				return errors.WithStackTrace(err)
			}

			logging.Debugf("Deleted api mapping: %s", *apiMapping.ApiMappingId)
		}
	}

	return nil
}

type TooManyApiGatewayV2Err struct{}

func (err TooManyApiGatewayV2Err) Error() string {
	return "Too many Api Gateways requested at once."
}
