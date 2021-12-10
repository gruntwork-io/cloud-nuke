package aws

import (
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func getAllKmsUserKeys(session *session.Session, batchSize int, excludeAfter time.Time) ([]*string, error) {
	svc := kms.New(session)
	var kmsIds []*string
	input := &kms.ListKeysInput{
		Limit: aws.Int64(int64(batchSize)),
	}
	listPage := 1
	err := svc.ListKeysPages(input, func(page *kms.ListKeysOutput, lastPage bool) bool {
		logging.Logger.Debugf("Loading User Key from page %d", listPage)

		wg := new(sync.WaitGroup)
		wg.Add(len(page.Keys))
		keyChans := make([]chan *kms.KeyListEntry, len(page.Keys))
		errChans := make([]chan error, len(page.Keys))
		for i, key := range page.Keys {
			keyChans[i] = make(chan *kms.KeyListEntry, 1)
			errChans[i] = make(chan error, 1)
			go shouldIncludeKmsUserKey(wg, svc, key, excludeAfter, keyChans[i], errChans[i])
		}
		wg.Wait()

		// collect errors
		var allErrs *multierror.Error
		for _, errChan := range errChans {
			if err := <-errChan; err != nil {
				allErrs = multierror.Append(allErrs, err)
				logging.Logger.Errorf("[Failed] %s", err)
			}
		}
		// stop in case of errors
		if allErrs.ErrorOrNil() != nil {
			return false
		}

		// collect keys
		for _, keyChan := range keyChans {
			if key := <-keyChan; key != nil {
				kmsIds = append(kmsIds, key.KeyId)
			}
		}

		listPage++
		return true
	})

	return kmsIds, errors.WithStackTrace(err)
}

func shouldIncludeKmsUserKey(wg *sync.WaitGroup, svc *kms.KMS, key *kms.KeyListEntry, excludeAfter time.Time, keyChan chan *kms.KeyListEntry, errChan chan error) {
	defer wg.Done()
	// additional request to describe key and get information about creation date, removal status
	details, err := svc.DescribeKey(&kms.DescribeKeyInput{KeyId: key.KeyId})
	if err != nil {
		errChan <- err
		keyChan <- nil
		return
	}
	metadata := details.KeyMetadata
	// evaluate only user keys
	if *metadata.KeyManager != kms.KeyManagerTypeCustomer {
		keyChan <- nil
		errChan <- nil
		return
	}
	// skip keys already scheduled for removal
	if metadata.DeletionDate != nil {
		keyChan <- nil
		errChan <- nil
		return
	}
	if metadata.PendingDeletionWindowInDays != nil {
		keyChan <- nil
		errChan <- nil
		return
	}
	var referenceTime = *metadata.CreationDate
	if referenceTime.After(excludeAfter) {
		keyChan <- nil
		errChan <- nil
		return
	}
	// put key in channel to be considered for removal
	keyChan <- key
	errChan <- nil
}

func nukeAllCustomerManagedKmsKeys(session *session.Session, keyIds []*string) error {
	region := aws.StringValue(session.Config.Region)
	if len(keyIds) == 0 {
		logging.Logger.Infof("No Customer Keys to nuke in region %s", region)
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
