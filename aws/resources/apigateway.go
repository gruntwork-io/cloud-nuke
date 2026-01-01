package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// ApiGatewayAPI defines the interface for API Gateway (v1) operations.
type ApiGatewayAPI interface {
	GetRestApis(ctx context.Context, params *apigateway.GetRestApisInput, optFns ...func(*apigateway.Options)) (*apigateway.GetRestApisOutput, error)
	GetStages(ctx context.Context, params *apigateway.GetStagesInput, optFns ...func(*apigateway.Options)) (*apigateway.GetStagesOutput, error)
	GetDomainNames(ctx context.Context, params *apigateway.GetDomainNamesInput, optFns ...func(*apigateway.Options)) (*apigateway.GetDomainNamesOutput, error)
	GetBasePathMappings(ctx context.Context, params *apigateway.GetBasePathMappingsInput, optFns ...func(*apigateway.Options)) (*apigateway.GetBasePathMappingsOutput, error)
	DeleteBasePathMapping(ctx context.Context, params *apigateway.DeleteBasePathMappingInput, optFns ...func(*apigateway.Options)) (*apigateway.DeleteBasePathMappingOutput, error)
	DeleteClientCertificate(ctx context.Context, params *apigateway.DeleteClientCertificateInput, optFns ...func(*apigateway.Options)) (*apigateway.DeleteClientCertificateOutput, error)
	DeleteRestApi(ctx context.Context, params *apigateway.DeleteRestApiInput, optFns ...func(*apigateway.Options)) (*apigateway.DeleteRestApiOutput, error)
}

// NewApiGateway creates a new ApiGateway resource using the generic resource pattern.
func NewApiGateway() AwsResource {
	return NewAwsResource(&resource.Resource[ApiGatewayAPI]{
		ResourceTypeName: "apigateway",
		BatchSize:        10,
		InitClient: func(r *resource.Resource[ApiGatewayAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for ApiGateway client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = apigateway.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.APIGateway
		},
		Lister: listApiGateways,
		Nuker:  deleteApiGateways,
	})
}

// listApiGateways retrieves all API Gateway (v1) REST APIs that match the config filters.
func listApiGateways(ctx context.Context, client ApiGatewayAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	result, err := client.GetRestApis(ctx, &apigateway.GetRestApisInput{})
	if err != nil {
		return []*string{}, errors.WithStackTrace(err)
	}

	var IDs []*string
	for _, api := range result.Items {
		if cfg.ShouldInclude(config.ResourceValue{
			Name: api.Name,
			Time: api.CreatedDate,
		}) {
			IDs = append(IDs, api.Id)
		}
	}

	return IDs, nil
}

// deleteApiGateways deletes the provided API Gateway (v1) REST APIs.
func deleteApiGateways(ctx context.Context, client ApiGatewayAPI, scope resource.Scope, resourceType string, identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No API Gateways (v1) to nuke in region %s", scope.Region)
		return nil
	}

	if len(identifiers) > 100 {
		logging.Errorf("Nuking too many API Gateways (v1) at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyApiGatewayErr{}
	}

	// There is no bulk delete Api Gateway API, so we delete the batch of gateways concurrently using goroutines
	logging.Debugf("Deleting Api Gateways (v1) in region %s", scope.Region)
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, apigwID := range identifiers {
		errChans[i] = make(chan error, 1)
		go deleteApiGatewayAsync(ctx, client, scope.Region, wg, errChans[i], apigwID)
	}
	wg.Wait()

	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Debugf("[Failed] %s", err)
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}

	return nil
}

func getAttachedStageClientCerts(ctx context.Context, client ApiGatewayAPI, apigwID *string) ([]*string, error) {
	var clientCerts []*string

	// remove the client certificate attached with the stages
	stages, err := client.GetStages(ctx, &apigateway.GetStagesInput{
		RestApiId: apigwID,
	})

	if err != nil {
		return nil, err
	}
	// get the stages attached client certificates
	for _, stage := range stages.Item {
		if stage.ClientCertificateId == nil {
			logging.Debugf("Skipping certyficate for stage %s, certyficate ID is nil", *stage.StageName)
			continue
		}
		clientCerts = append(clientCerts, stage.ClientCertificateId)
	}
	return clientCerts, nil
}

func removeAttachedClientCertificates(ctx context.Context, client ApiGatewayAPI, clientCerts []*string) error {
	for _, cert := range clientCerts {
		logging.Debugf("Deleting Client Certificate %s", *cert)
		_, err := client.DeleteClientCertificate(ctx, &apigateway.DeleteClientCertificateInput{
			ClientCertificateId: cert,
		})
		if err != nil {
			logging.Errorf("[Failed] Error deleting Client Certificate %s", *cert)
			return err
		}
	}
	return nil
}

func deleteApiGatewayAsync(
	ctx context.Context, client ApiGatewayAPI, region string,
	wg *sync.WaitGroup, errChan chan error, apigwID *string,
) {
	defer wg.Done()

	var err error

	// Defer error reporting, channel sending, and logging
	defer func() {
		errChan <- err

		// Record status of this resource
		e := report.Entry{
			Identifier:   *apigwID,
			ResourceType: "APIGateway (v1)",
			Error:        err,
		}
		report.Record(e)

		if err == nil {
			logging.Debugf("[OK] API Gateway (v1) %s deleted in %s", aws.ToString(apigwID), region)
		} else {
			logging.Debugf("[Failed] Error deleting API Gateway (v1) %s in %s", aws.ToString(apigwID), region)
		}
	}()

	// get the attached client certificates
	var clientCerts []*string
	clientCerts, err = getAttachedStageClientCerts(ctx, client, apigwID)
	if err != nil {
		return
	}

	// Check if the API Gateway has any associated API mappings.
	// If so, remove them before deleting the API Gateway.
	err = deleteAssociatedApiMappingsV1(ctx, client, []*string{apigwID})
	if err != nil {
		return
	}

	// delete the API Gateway
	input := &apigateway.DeleteRestApiInput{RestApiId: apigwID}
	_, err = client.DeleteRestApi(ctx, input)
	if err != nil {
		return
	}

	// When the rest-api endpoint delete successfully, then remove attached client certs
	err = removeAttachedClientCertificates(ctx, client, clientCerts)
}

// deleteAssociatedApiMappingsV1 deletes API mappings for API Gateway v1.
// Named with V1 suffix to avoid conflict with similar function in apigatewayv2.go.
func deleteAssociatedApiMappingsV1(ctx context.Context, client ApiGatewayAPI, identifiers []*string) error {
	// Convert identifiers to map to check if identifier is in list
	identifierMap := make(map[string]struct{})
	for _, identifier := range identifiers {
		identifierMap[*identifier] = struct{}{}
	}

	domainNames, err := client.GetDomainNames(ctx, &apigateway.GetDomainNamesInput{})
	if err != nil {
		logging.Debugf("Failed to get domain names: %s", err)
		return errors.WithStackTrace(err)
	}

	logging.Debugf("Found %d domain name(s)", len(domainNames.Items))
	for _, domain := range domainNames.Items {

		apiMappings, err := client.GetBasePathMappings(ctx, &apigateway.GetBasePathMappingsInput{
			DomainName: domain.DomainName,
		})

		if err != nil {
			logging.Debugf("Failed to get base path mappings for domain %s: %v", *domain.DomainName, err)
			return errors.WithStackTrace(err)
		}
		logging.Debugf("Found %d base path mappings for domain %s", len(apiMappings.Items), *domain.DomainName)

		for _, mapping := range apiMappings.Items {

			if mapping.RestApiId == nil {
				continue
			}

			if _, found := identifierMap[*mapping.RestApiId]; !found {
				continue
			}

			logging.Debugf("Deleting base path mapping for API %s on domain %s", *mapping.RestApiId, *domain.DomainName)

			_, err := client.DeleteBasePathMapping(ctx, &apigateway.DeleteBasePathMappingInput{
				DomainName: domain.DomainName,
				BasePath:   mapping.BasePath,
			})
			if err != nil {
				logging.Debugf("Failed to delete base path mapping for API %s: %v", *mapping.RestApiId, err)
				return errors.WithStackTrace(err)
			}

			logging.Debugf("Successfully deleted base path mapping for API %s", *mapping.RestApiId)
		}
	}

	logging.Debug("Completed deletion of matching API mappings.")
	return nil
}

// TooManyApiGatewayErr is returned when too many API Gateways are requested at once.
type TooManyApiGatewayErr struct{}

func (err TooManyApiGatewayErr) Error() string {
	return "Too many Api Gateways requested at once."
}
