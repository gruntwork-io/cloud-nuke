package aws

import (
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func getAllAPIGateways(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := apigateway.New(session)

	result, err := svc.GetRestApis(&apigateway.GetRestApisInput{})
	if err != nil {
		return []*string{}, errors.WithStackTrace(err)
	}
	Ids := []*string{}
	for _, apigateway := range result.Items {
		if shouldIncludeAPIGateway(apigateway, excludeAfter, configObj) {
			Ids = append(Ids, apigateway.Id)
		}
	}

	return Ids, nil
}

func shouldIncludeAPIGateway(apigw *apigateway.RestApi, excludeAfter time.Time, configObj config.Config) bool {
	if apigw == nil {
		return false
	}

	if apigw.CreatedDate != nil {
		if excludeAfter.Before(aws.TimeValue(apigw.CreatedDate)) {
			return false
		}
	}

	return config.ShouldInclude(
		aws.StringValue(apigw.Name),
		configObj.APIGateway.IncludeRule.NamesRegExp,
		configObj.APIGateway.ExcludeRule.NamesRegExp,
	)
}

func nukeAllAPIGateways(session *session.Session, identifiers []*string) error {
	region := aws.StringValue(session.Config.Region)

	svc := apigateway.New(session)

	if len(identifiers) == 0 {
		logging.Logger.Infof("No API Gateways (v1) to nuke in region %s", region)
	}

	if len(identifiers) > 100 {
		logging.Logger.Errorf("Nuking too many API Gateways (v1) at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyApiGatewayErr{}
	}

	// There is no bulk delete Api Gateway API, so we delete the batch of gateways concurrently using goroutines
	logging.Logger.Infof("Deleting Api Gateways (v1) in region %s", region)
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, apigwID := range identifiers {
		errChans[i] = make(chan error, 1)
		go deleteApiGatewayAsync(wg, errChans[i], svc, apigwID, region)
	}
	wg.Wait()

	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Logger.Errorf("[Failed] %s", err)
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}
	return nil
}

func deleteApiGatewayAsync(wg *sync.WaitGroup, errChan chan error, svc *apigateway.APIGateway, apigwID *string, region string) {
	defer wg.Done()

	input := &apigateway.DeleteRestApiInput{RestApiId: apigwID}
	_, err := svc.DeleteRestApi(input)
	errChan <- err

	if err == nil {
		logging.Logger.Infof("[OK] API Gateway (v1) %s deleted in %s", aws.StringValue(apigwID), region)
	} else {
		logging.Logger.Errorf("[Failed] Error deleting API Gateway (v1) %s in %s", aws.StringValue(apigwID), region)
	}
}
