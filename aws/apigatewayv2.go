package aws

import (
	"sync"

	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigatewayv2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func (gw ApiGatewayV2) getAll(configObj config.Config) ([]*string, error) {
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

func (gw ApiGatewayV2) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Logger.Debugf("No API Gateways (v2) to nuke in region %s", gw.Region)
	}

	if len(identifiers) > 100 {
		logging.Logger.Debugf("Nuking too many API Gateways (v2) at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyApiGatewayV2Err{}
	}

	// There is no bulk delete Api Gateway API, so we delete the batch of gateways concurrently using goroutines
	logging.Logger.Debugf("Deleting Api Gateways (v2) in region %s", gw.Region)
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
			logging.Logger.Debugf("[Failed] %s", err)
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

func (gw ApiGatewayV2) deleteAsync(wg *sync.WaitGroup, errChan chan error, apiId *string) {
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
		logging.Logger.Debugf("[OK] API Gateway (v2) %s deleted in %s", aws.StringValue(apiId), gw.Region)
	} else {
		logging.Logger.Debugf("[Failed] Error deleting API Gateway (v2) %s in %s", aws.StringValue(apiId), gw.Region)
	}
}
