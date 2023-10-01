package resources

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/service/apigatewayv2/apigatewayv2iface"
	"github.com/pterm/pterm"
	"sync"

	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigatewayv2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func (gw *ApiGatewayV2) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	output, err := gw.Client.GetApis(&apigatewayv2.GetApisInput{})
	if err != nil {
		return []*string{}, errors.WithStackTrace(err)
	}

	Ids := []*string{}
	for _, restapi := range output.Items {
		if configObj.APIGatewayV2.ShouldInclude(config.ResourceValue{
			Time: restapi.CreatedDate,
			Name: restapi.Name,
		}) {
			Ids = append(Ids, restapi.ApiId)
		}
	}

	return Ids, nil
}

func (gw *ApiGatewayV2) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		pterm.Debug.Println(fmt.Sprintf("No API Gateways (v2) to nuke in region %s", gw.Region))
	}

	if len(identifiers) > 100 {
		pterm.Debug.Println(fmt.Sprintf(
			"Nuking too many API Gateways (v2) at once (100): halting to avoid hitting AWS API rate limiting"))
		return TooManyApiGatewayV2Err{}
	}

	err := deleteAssociatedApiMappings(gw.Client, identifiers)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	// There is no bulk delete Api Gateway API, so we delete the batch of gateways concurrently using goroutines
	pterm.Debug.Println(fmt.Sprintf("Deleting Api Gateways (v2) in region %s", gw.Region))
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, apigwID := range identifiers {
		errChans[i] = make(chan error, 1)
		go gw.deleteAsync(wg, errChans[i], apigwID)
	}
	wg.Wait()

	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking API Gateway V2",
			}, map[string]interface{}{
				"region": gw.Region,
			})
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}
	return nil
}

func (gw *ApiGatewayV2) deleteAsync(wg *sync.WaitGroup, errChan chan error, apiId *string) {
	defer wg.Done()

	input := &apigatewayv2.DeleteApiInput{ApiId: apiId}
	_, err := gw.Client.DeleteApi(input)
	errChan <- err

	// Record status of this resource
	e := report.Entry{
		Identifier:   *apiId,
		ResourceType: "APIGateway (v2)",
		Error:        err,
	}
	report.Record(e)

	if err == nil {
		pterm.Debug.Println(fmt.Sprintf("Successfully deleted API Gateway (v2) %s in %s", aws.StringValue(apiId), gw.Region))
	} else {
		pterm.Debug.Println(fmt.Sprintf("Failed to delete API Gateway (v2) %s in %s", aws.StringValue(apiId), gw.Region))
	}
}

func deleteAssociatedApiMappings(
	client apigatewayv2iface.ApiGatewayV2API, identifiers []*string) error {
	// Convert identifiers to map to check if identifier is in list
	identifierMap := make(map[string]bool)
	for _, identifier := range identifiers {
		identifierMap[*identifier] = true
	}

	domainNames, err := client.GetDomainNames(&apigatewayv2.GetDomainNamesInput{})
	if err != nil {
		pterm.Debug.Println(fmt.Sprintf("Failed to get domain names: %s", err))
		return errors.WithStackTrace(err)
	}

	pterm.Debug.Println(fmt.Sprintf("Found %d domain names", len(domainNames.Items)))
	for _, domainName := range domainNames.Items {
		apiMappings, err := client.GetApiMappings(&apigatewayv2.GetApiMappingsInput{
			DomainName: domainName.DomainName,
		})
		if err != nil {
			pterm.Debug.Println(fmt.Sprintf("Failed to get api mappings: %s", err))
			return errors.WithStackTrace(err)
		}

		for _, apiMapping := range apiMappings.Items {
			if _, ok := identifierMap[*apiMapping.ApiId]; !ok {
				continue
			}

			_, err := client.DeleteApiMapping(&apigatewayv2.DeleteApiMappingInput{
				ApiMappingId: apiMapping.ApiMappingId,
				DomainName:   domainName.DomainName,
			})
			if err != nil {
				pterm.Debug.Println(fmt.Sprintf("Failed to delete api mapping: %s", err))
				return errors.WithStackTrace(err)
			}

			pterm.Debug.Println(fmt.Sprintf("Deleted api mapping: %s", *apiMapping.ApiMappingId))
		}
	}

	return nil
}
