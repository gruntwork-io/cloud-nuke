package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func (gateway *ApiGateway) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := gateway.Client.GetRestApis(&apigateway.GetRestApisInput{})
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
	stages, err := gateway.Client.GetStages(&apigateway.GetStagesInput{
		RestApiId: apigwID,
	})

	if err != nil {
		return clientCerts, err
	}
	// get the stages attached client certificates
	for _, stage := range stages.Item {
		clientCerts = append(clientCerts, stage.ClientCertificateId)
	}
	return clientCerts, nil
}

func (gateway *ApiGateway) removeAttachedClientCertificates(clientCerts []*string) error {

	for _, cert := range clientCerts {
		logging.Debugf("Deleting Client Certificate %s", *cert)
		_, err := gateway.Client.DeleteClientCertificate(&apigateway.DeleteClientCertificateInput{
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
	wg *sync.WaitGroup, errChan chan error, apigwID *string) {
	defer wg.Done()

	// get the attached client certificates
	clientCerts, err := gateway.getAttachedStageClientCerts(apigwID)

	input := &apigateway.DeleteRestApiInput{RestApiId: apigwID}
	_, err = gateway.Client.DeleteRestApi(input)

	// When the rest-api endpoint delete successfully, then remove attached client certs
	if err == nil {
		err = gateway.removeAttachedClientCertificates(clientCerts)
	}

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
		logging.Debugf("["+
			"OK] API Gateway (v1) %s deleted in %s", aws.StringValue(apigwID), gateway.Region)
		return
	}

	logging.Debugf(
		"[Failed] Error deleting API Gateway (v1) %s in %s", aws.StringValue(apigwID), gateway.Region)
}
