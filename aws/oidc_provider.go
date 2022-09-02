package aws

import (
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// oidcProvider is an internal struct to collect the necessary information we need to filter in the OIDC Providers that
// should be deleted. This exists because no struct in the AWS SDK represents all the information collected here.
type oidcProvider struct {
	ARN         *string
	CreateTime  *time.Time
	ProviderURL *string
}

// getAllOIDCProviders will list all the OpenID Connect Providers in an account, filtering out those that do not match
// the requested rules (older-than and config file settings). Note that since the list API does not return the necessary
// information to implement the filters, we use goroutines to asynchronously and concurrently fetch the details for all
// the providers that are found in the account.
func getAllOIDCProviders(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := iam.New(session)

	output, err := svc.ListOpenIDConnectProviders(&iam.ListOpenIDConnectProvidersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	providerARNs := []*string{}
	for _, provider := range output.OpenIDConnectProviderList {
		providerARNs = append(providerARNs, provider.Arn)
	}
	providers, err := getAllOIDCProviderDetails(svc, providerARNs)
	if err != nil {
		return nil, err
	}

	providerARNsToDelete := []*string{}
	for _, provider := range providers {
		if shouldIncludeOIDCProvider(provider, excludeAfter, configObj) {
			providerARNsToDelete = append(providerARNsToDelete, provider.ARN)
		}
	}
	return providerARNsToDelete, nil
}

// getAllOIDCProviderDetails fetches the details of the given list of OpenID Connect Providers so that we can make
// informed decisions about which ones should be included in the nuking procedure.
func getAllOIDCProviderDetails(svc *iam.IAM, providerARNs []*string) ([]oidcProvider, error) {
	numRetrieving := len(providerARNs)

	// Schedule goroutines to retrieve the provider details async.
	wg := new(sync.WaitGroup)
	wg.Add(numRetrieving)
	resultChans := make([]chan *oidcProvider, numRetrieving)
	errChans := make([]chan error, numRetrieving)
	for i, providerARN := range providerARNs {
		resultChans[i] = make(chan *oidcProvider, 1)
		errChans[i] = make(chan error, 1)
		go getOIDCProviderDetailAsync(wg, resultChans[i], errChans[i], svc, providerARN)
	}
	wg.Wait()

	// Collect errors, if any.
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return nil, errors.WithStackTrace(finalErr)
	}

	// Collect results, if any.
	var allResults []oidcProvider
	for _, resultChan := range resultChans {
		if result := <-resultChan; result != nil {
			allResults = append(allResults, *result)
		}
	}

	return allResults, nil
}

// shouldIncludeOIDCProvider determines whether or not a given OpenID Connect Provider should be nuked based on the
// given constraints.
func shouldIncludeOIDCProvider(provider oidcProvider, excludeAfter time.Time, configObj config.Config) bool {
	if excludeAfter.Before(aws.TimeValue(provider.CreateTime)) {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(provider.ProviderURL),
		configObj.OIDCProvider.IncludeRule.NamesRegExp,
		configObj.OIDCProvider.ExcludeRule.NamesRegExp,
	)
}

// getOIDCProviderDetailAsync is a routine for fetching the details of a single OpenID Connect Provider. This function
// is designed to be called in a goroutine.
func getOIDCProviderDetailAsync(wg *sync.WaitGroup, resultChan chan *oidcProvider, errChan chan error, svc *iam.IAM, providerARN *string) {
	defer wg.Done()

	resp, err := svc.GetOpenIDConnectProvider(&iam.GetOpenIDConnectProviderInput{OpenIDConnectProviderArn: providerARN})
	if err != nil {
		// If we get a 404, meaning the OIDC Provider was deleted between retrieving it with list and detail fetching,
		// we ignore the error and return nothing.
		aerr, ok := err.(awserr.Error)
		if ok && aerr.Code() == iam.ErrCodeNoSuchEntityException {
			resultChan <- nil
			errChan <- nil
			return
		}

		// For all other errors, bubble the error
		resultChan <- nil
		errChan <- errors.WithStackTrace(err)
		return
	}

	provider := oidcProvider{
		ARN:         providerARN,
		CreateTime:  resp.CreateDate,
		ProviderURL: resp.Url,
	}
	resultChan <- &provider
	errChan <- nil
}

// nukeAllOIDCProviders deletes all the given OpenID Connect Providers from the account.
func nukeAllOIDCProviders(session *session.Session, identifiers []*string) error {
	svc := iam.New(session)

	if len(identifiers) == 0 {
		logging.Logger.Infof("No OIDC Providers to nuke")
		return nil
	}

	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on OIDCProviders.MaxBatchSize, however we add a guard here to warn users when the batching fails and has a
	// chance of throttling AWS. Since we concurrently make one call for each identifier, we pick 100 for the limit here
	// because many APIs in AWS have a limit of 100 requests per second.
	if len(identifiers) > 100 {
		logging.Logger.Errorf("Nuking too many OIDC Providers at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyOIDCProvidersErr{}
	}

	// There is no bulk delete OIDC Provider API, so we delete the batch of nat gateways concurrently using go routines.
	logging.Logger.Infof("Deleting OIDC Providers")
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, providerARN := range identifiers {
		errChans[i] = make(chan error, 1)
		go deleteOIDCProviderAsync(wg, errChans[i], svc, providerARN)
	}
	wg.Wait()

	// Collect all the errors from the async delete calls into a single error struct.
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

	for _, providerARN := range identifiers {
		logging.Logger.Infof("[OK] OIDC Provider %s was deleted", aws.StringValue(providerARN))
	}
	return nil
}

// deleteOIDCProviderAsync deletes the provided OIDC Provider asynchronously in a goroutine, using wait groups for
// concurrency control and a return channel for errors.
func deleteOIDCProviderAsync(wg *sync.WaitGroup, errChan chan error, svc *iam.IAM, providerARN *string) {
	defer wg.Done()

	_, err := svc.DeleteOpenIDConnectProvider(&iam.DeleteOpenIDConnectProviderInput{OpenIDConnectProviderArn: providerARN})
	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.StringValue(providerARN),
		ResourceType: "OIDC Provider",
		Error:        err,
	}
	report.Record(e)

	errChan <- err
}

// Custom errors

type TooManyOIDCProvidersErr struct{}

func (err TooManyOIDCProvidersErr) Error() string {
	return "Too many OIDC Providers requested at once."
}
