package aws

import (
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	multierror "github.com/hashicorp/go-multierror"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func getAllSecretsManagerSecrets(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := secretsmanager.New(session)

	allSecrets := []*string{}
	input := &secretsmanager.ListSecretsInput{}
	err := svc.ListSecretsPages(
		input,
		func(page *secretsmanager.ListSecretsOutput, lastPage bool) bool {
			for _, secret := range page.SecretList {
				if shouldIncludeSecret(secret, excludeAfter, configObj) {
					allSecrets = append(allSecrets, secret.ARN)
				}
			}
			return !lastPage
		},
	)
	return allSecrets, errors.WithStackTrace(err)
}

func shouldIncludeSecret(secret *secretsmanager.SecretListEntry, excludeAfter time.Time, configObj config.Config) bool {
	if secret == nil {
		return false
	}

	// reference time for excludeAfter is last accessed time, unless it was never accessed in which created time is
	// used.
	var referenceTime time.Time
	if secret.LastAccessedDate == nil {
		referenceTime = *secret.CreatedDate
	} else {
		referenceTime = *secret.LastAccessedDate
	}
	if excludeAfter.Before(referenceTime) {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(secret.Name),
		configObj.SecretsManagerSecrets.IncludeRule.NamesRegExp,
		configObj.SecretsManagerSecrets.ExcludeRule.NamesRegExp,
	)
}

func nukeAllSecretsManagerSecrets(session *session.Session, identifiers []*string) error {
	region := aws.StringValue(session.Config.Region)

	svc := secretsmanager.New(session)

	if len(identifiers) == 0 {
		logging.Logger.Debugf("No Secrets Manager Secrets to nuke in region %s", region)
		return nil
	}

	// There is no bulk delete secrets API, so we delete the batch of secrets concurrently using go routines.
	logging.Logger.Debugf("Deleting Secrets Manager secrets in region %s", region)
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, secretID := range identifiers {
		errChans[i] = make(chan error, 1)
		go deleteSecretAsync(wg, errChans[i], svc, secretID)
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
	return errors.WithStackTrace(allErrs.ErrorOrNil())
}

// deleteSecretAsync deletes the provided secrets manager secret. Intended to be run in a goroutine, using wait groups
// and a return channel for errors.
func deleteSecretAsync(wg *sync.WaitGroup, errChan chan error, svc *secretsmanager.SecretsManager, secretID *string) {
	defer wg.Done()

	input := &secretsmanager.DeleteSecretInput{
		ForceDeleteWithoutRecovery: aws.Bool(true),
		SecretId:                   secretID,
	}
	_, err := svc.DeleteSecret(input)

	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.StringValue(secretID),
		ResourceType: "Secrets Manager Secret",
		Error:        err,
	}
	report.Record(e)

	errChan <- err
}
