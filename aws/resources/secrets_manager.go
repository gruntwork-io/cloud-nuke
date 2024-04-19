package resources

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/hashicorp/go-multierror"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (sms *SecretsManagerSecrets) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	allSecrets := []*string{}
	input := &secretsmanager.ListSecretsInput{}
	err := sms.Client.ListSecretsPages(
		input,
		func(page *secretsmanager.ListSecretsOutput, lastPage bool) bool {
			for _, secret := range page.SecretList {
				if shouldIncludeSecret(secret, configObj) {
					allSecrets = append(allSecrets, secret.ARN)
				}
			}
			return !lastPage
		},
	)
	return allSecrets, errors.WithStackTrace(err)
}

func shouldIncludeSecret(secret *secretsmanager.SecretListEntry, configObj config.Config) bool {
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

	return configObj.SecretsManagerSecrets.ShouldInclude(config.ResourceValue{
		Time: &referenceTime,
		Name: secret.Name,
	})
}

func (sms *SecretsManagerSecrets) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Secrets Manager Secrets to nuke in region %s", sms.Region)
		return nil
	}

	// There is no bulk delete secrets API, so we delete the batch of secrets concurrently using go routines.
	logging.Debugf("Deleting Secrets Manager secrets in region %s", sms.Region)
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, secretID := range identifiers {
		errChans[i] = make(chan error, 1)
		go sms.deleteAsync(wg, errChans[i], secretID)
	}
	wg.Wait()

	// Collect all the errors from the async delete calls into a single error struct.
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Errorf("[Failed] %s", err)
		}
	}
	return errors.WithStackTrace(allErrs.ErrorOrNil())
}

// deleteAsync deletes the provided secrets manager secret. Intended to be run in a goroutine, using wait groups
// and a return channel for errors.
func (sms *SecretsManagerSecrets) deleteAsync(wg *sync.WaitGroup, errChan chan error, secretID *string) {
	defer wg.Done()

	// If this region's secret is primary, and it has replicated secrets, remove replication first.
	// Get replications
	secret, err := sms.Client.DescribeSecret(&secretsmanager.DescribeSecretInput{
		SecretId: secretID,
	})

	// Delete replications
	if len(secret.ReplicationStatus) > 0 {
		replicationRegion := make([]*string, 0)

		// Get replicas' region
		for _, replicationStatus := range secret.ReplicationStatus {
			replicationRegion = append(replicationRegion, replicationStatus.Region)
		}

		_, err = sms.Client.RemoveRegionsFromReplication(&secretsmanager.RemoveRegionsFromReplicationInput{
			SecretId:             secretID,
			RemoveReplicaRegions: replicationRegion,
		})
	}

	input := &secretsmanager.DeleteSecretInput{
		ForceDeleteWithoutRecovery: aws.Bool(true),
		SecretId:                   secretID,
	}
	_, err = sms.Client.DeleteSecret(input)

	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.StringValue(secretID),
		ResourceType: "Secrets Manager Secret",
		Error:        err,
	}
	report.Record(e)

	errChan <- err
}
