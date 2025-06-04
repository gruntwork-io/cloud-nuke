package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func (gateway *ApiGateway) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := gateway.Client.GetRestApis(c, &apigateway.GetRestApisInput{})
	if err != nil {
		return []*string{}, errors.WithStackTrace(err)
	}

	var IDs []*string
	for _, api := range result.Items {
		if configObj.APIGateway.ShouldInclude(config.ResourceValue{
			Name: api.Name,
			Time: api.CreatedDate,
		}) {
			IDs = append(IDs, api.Id)
		}
	}

	return IDs, nil
}

func (gateway *ApiGateway) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No API Gateways (v1) to nuke in region %s", gateway.Region)
	}

	if len(identifiers) > 100 {
		logging.Errorf("Nuking too many API Gateways (v1) at once (100): " +
			"halting to avoid hitting AWS API rate limiting")
		return TooManyApiGatewayErr{}
	}

	// There is no bulk delete Api Gateway API, so we delete the batch of gateways concurrently using goroutines
	logging.Debugf("Deleting Api Gateways (v1) in region %s", gateway.Region)
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, apigwID := range identifiers {
		errChans[i] = make(chan error, 1)
		go gateway.nukeAsync(wg, errChans[i], apigwID)
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

func (gateway *ApiGateway) getAttachedStageClientCerts(apigwID *string) ([]*string, error) {
	var clientCerts []*string

	// remove the client certificate attached with the stages
	stages, err := gateway.Client.GetStages(gateway.Context, &apigateway.GetStagesInput{
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

func (gateway *ApiGateway) removeAttachedClientCertificates(clientCerts []*string) error {

	for _, cert := range clientCerts {
		logging.Debugf("Deleting Client Certificate %s", *cert)
		_, err := gateway.Client.DeleteClientCertificate(gateway.Context, &apigateway.DeleteClientCertificateInput{
			ClientCertificateId: cert,
		})
		if err != nil {
			logging.Errorf("[Failed] Error deleting Client Certificate %s", *cert)
			return err
		}
	}
	return nil
}

func (gateway *ApiGateway) nukeAsync(
	wg *sync.WaitGroup, errChan chan error, apigwID *string,
) {
	defer wg.Done()

	var err error

	// Why defer?
	// Defer error reporting, channel sending, and logging to ensure they run
	// after function execution completes, regardless of success or failure.
	// This ensures consistent reporting, prevents missed logs, and avoids
	// duplicated code paths for error/success handling.
	//
	// See: https://go.dev/ref/spec#Defer_statements
	defer func() {
		// send the error data to channel
		errChan <- err

		// Record status of this resource
		e := report.Entry{
			Identifier:   *apigwID,
			ResourceType: "APIGateway (v1)",
			Error:        err,
		}
		report.Record(e)

		if err == nil {
			logging.Debugf("[OK] API Gateway (v1) %s deleted in %s", aws.ToString(apigwID), gateway.Region)
		} else {
			logging.Debugf("[Failed] Error deleting API Gateway (v1) %s in %s", aws.ToString(apigwID), gateway.Region)
		}
	}()

	// get the attached client certificates
	var clientCerts []*string
	clientCerts, err = gateway.getAttachedStageClientCerts(apigwID)
	if err != nil {
		return
	}

	// Check if the API Gateway has any associated API mappings.
	// If so, remove them before deleting the API Gateway.
	err = gateway.deleteAssociatedApiMappings(context.Background(), []*string{apigwID})
	if err != nil {
		return
	}

	// delete the API Gateway
	input := &apigateway.DeleteRestApiInput{RestApiId: apigwID}
	_, err = gateway.Client.DeleteRestApi(gateway.Context, input)
	if err != nil {
		return
	}

	// When the rest-api endpoint delete successfully, then remove attached client certs
	err = gateway.removeAttachedClientCertificates(clientCerts)
}

func (gateway *ApiGateway) deleteAssociatedApiMappings(ctx context.Context, identifiers []*string) error {
	// Convert identifiers to map to check if identifier is in list
	identifierMap := make(map[string]struct{})
	for _, identifier := range identifiers {
		identifierMap[*identifier] = struct{}{}
	}

	domainNames, err := gateway.Client.GetDomainNames(ctx, &apigateway.GetDomainNamesInput{})
	if err != nil {
		logging.Debugf("Failed to get domain names: %s", err)
		return errors.WithStackTrace(err)
	}

	logging.Debugf("Found %d domain name(s)", len(domainNames.Items))
	for _, domain := range domainNames.Items {

		apiMappings, err := gateway.Client.GetBasePathMappings(ctx, &apigateway.GetBasePathMappingsInput{
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

			_, err := gateway.Client.DeleteBasePathMapping(ctx, &apigateway.DeleteBasePathMappingInput{
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
