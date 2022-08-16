package aws

import (
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigatewayv2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func getAllAPIGatewaysV2(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := apigatewayv2.New(session)

	output, err := svc.GetApis(&apigatewayv2.GetApisInput{})
	if err != nil {
		return []*string{}, errors.WithStackTrace(err)
	}
	Ids := []*string{}
	for _, restapi := range output.Items {
		if shouldIncludeAPIGatewayV2(restapi, excludeAfter, configObj) {
			Ids = append(Ids, restapi.ApiId)
		}
	}

	return Ids, nil
}

func shouldIncludeAPIGatewayV2(api *apigatewayv2.Api, excludeAfter time.Time, configObj config.Config) bool {
	if api == nil {
		return false
	}

	if api.CreatedDate != nil {
		if excludeAfter.Before(aws.TimeValue(api.CreatedDate)) {
			return false
		}
	}

	return config.ShouldInclude(
		aws.StringValue(api.Name),
		configObj.APIGateway.IncludeRule.NamesRegExp,
		configObj.APIGateway.ExcludeRule.NamesRegExp,
	)
}

func nukeAllAPIGatewaysV2(session *session.Session, identifiers []*string) error {
	region := aws.StringValue(session.Config.Region)

	svc := apigatewayv2.New(session)

	if len(identifiers) == 0 {
		logging.Logger.Infof("No API Gateways (v2) to nuke in region %s", region)
	}

	if len(identifiers) > 100 {
		logging.Logger.Errorf("Nuking too many API Gateways (v2) at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyApiGatewayV2Err{}
	}

	// There is no bulk delete Api Gateway API, so we delete the batch of gateways concurrently using goroutines
	logging.Logger.Infof("Deleting Api Gateways (v2) in region %s", region)
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, apigwID := range identifiers {
		errChans[i] = make(chan error, 1)
		go deleteApiGatewayAsyncV2(wg, errChans[i], svc, apigwID, region)
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

func deleteApiGatewayAsyncV2(wg *sync.WaitGroup, errChan chan error, svc *apigatewayv2.ApiGatewayV2, apiId *string, region string) {
	defer wg.Done()

	input := &apigatewayv2.DeleteApiInput{ApiId: apiId}
	_, err := svc.DeleteApi(input)
	errChan <- err

	if err == nil {
		logging.Logger.Infof("[OK] API Gateway (v1) %s deleted in %s", aws.StringValue(apiId), region)
	} else {
		logging.Logger.Errorf("[Failed] Error deleting API Gateway (v1) %s in %s", aws.StringValue(apiId), region)
	}
}
