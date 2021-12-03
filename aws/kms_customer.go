package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
	"sync"
	"time"
)

func getAllKmsUserKeys(session *session.Session, batchSize int, excludeAfter time.Time) ([]*string, error) {
	svc := kms.New(session)
	var kmsIds []*string
	input := &kms.ListKeysInput{
		Limit: aws.Int64(int64(batchSize)),
	}
	listPage := 1
	err := svc.ListKeysPages(input, func(page *kms.ListKeysOutput, lastPage bool) bool {
		logging.Logger.Debugf("Loading User Key %d", listPage)

		for _, key := range page.Keys {
			include, err := shouldIncludeKmsUserKey(svc, key, excludeAfter)
			if err != nil {
				logging.Logger.Errorf("Occured error during checking key %v", err)
				return false
			}
			if  include{
				kmsIds = append(kmsIds, key.KeyId)
			}
		}
		listPage++
		return true
	})

	return kmsIds, errors.WithStackTrace(err)
}

func shouldIncludeKmsUserKey(svc *kms.KMS, key *kms.KeyListEntry, excludeAfter time.Time) (bool, error) {
	details, err := svc.DescribeKey(&kms.DescribeKeyInput{KeyId: key.KeyId})
	if err != nil {
		return false, nil
	}
	metadata := details.KeyMetadata
	// evaluate only user keys
	if *metadata.KeyManager != kms.KeyManagerTypeCustomer {
		return false, err
	}
	// skip keys already scheduled for removal
	if metadata.DeletionDate != nil {
		return false, nil
	}
	if metadata.PendingDeletionWindowInDays != nil {
		return false, nil
	}
	var referenceTime = *metadata.CreationDate
	if referenceTime.After(excludeAfter) {
		return false, nil
	}
	return true, nil
}

func nukeAllCustomerManagedKmsKeys(session *session.Session, keyIds []*string) error {
	region := aws.StringValue(session.Config.Region)
	if len(keyIds) == 0 {
		logging.Logger.Info("No Customer Keys to nuke in region %s", region)
		return nil
	}

	// usage of go routines for parallel keys removal
	// https://docs.aws.amazon.com/sdk-for-go/api/service/kms/#KMS.ScheduleKeyDeletion
	logging.Logger.Infof("Deleting Keys secrets in region %s", region)
	svc := kms.New(session)
	wg := new(sync.WaitGroup)
	wg.Add(len(keyIds))
	errChans := make([]chan error, len(keyIds))
	for i, secretID := range keyIds {
		errChans[i] = make(chan error, 1)
		go requestKeyDeletion(wg, errChans[i], svc, secretID)
	}
	wg.Wait()

	// collect errors from each channel
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Logger.Errorf("[Failed] %s", err)
		}
	}
	return errors.WithStackTrace(allErrs.ErrorOrNil())
}

func requestKeyDeletion(wg *sync.WaitGroup, errChan chan error, svc *kms.KMS, key *string) {
	defer wg.Done()
	input := &kms.ScheduleKeyDeletionInput{KeyId: key, PendingWindowInDays: aws.Int64(int64(kmsRemovalWindow))}
	_, err := svc.ScheduleKeyDeletion(input)
	errChan <- err
}
