package aws

import (
	"sync"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/report"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func getAllKmsUserKeys(session *session.Session, batchSize int, excludeAfter time.Time, configObj config.Config) ([]*string, map[string][]string, error) {
	svc := kms.New(session)
	// collect all aliases for each key
	keyAliases, err := listKeyAliases(svc, batchSize)
	if err != nil {
		return nil, nil, errors.WithStackTrace(err)
	}
	// checking in parallel if keys can be considered for removal
	var wg sync.WaitGroup
	wg.Add(len(keyAliases))
	resultsChan := make([]chan *KmsCheckIncludeResult, len(keyAliases))
	id := 0
	for key, aliases := range keyAliases {
		resultsChan[id] = make(chan *KmsCheckIncludeResult, 1)
		go shouldIncludeKmsUserKey(&wg, resultsChan[id], svc, key, aliases, excludeAfter, configObj)
		id++
	}
	wg.Wait()

	var kmsIds []*string
	aliases := map[string][]string{}

	for _, channel := range resultsChan {
		result := <-channel
		if result.Error != nil {
			logging.Logger.Debugf("Can't read KMS key %s", result.Error)

			continue
		}
		if result.KeyId != "" {
			aliases[result.KeyId] = keyAliases[result.KeyId]
			kmsIds = append(kmsIds, &result.KeyId)
		}
	}
	return kmsIds, aliases, nil
}

// KmsCheckIncludeResult - structure used results of evaluation: not null KeyId - key should be included
type KmsCheckIncludeResult struct {
	KeyId string
	Error error
}

func shouldIncludeKmsUserKey(wg *sync.WaitGroup, resultsChan chan *KmsCheckIncludeResult, svc *kms.KMS, key string,
	aliases []string, excludeAfter time.Time, configObj config.Config,
) {
	defer wg.Done()
	includedByName := false
	// verify if key aliases matches configurations
	for _, alias := range aliases {
		v := config.ShouldInclude(alias, configObj.KMSCustomerKeys.IncludeRule.NamesRegExp,
			configObj.KMSCustomerKeys.ExcludeRule.NamesRegExp)
		if v {
			includedByName = true

			break
		}
	}
	if !includedByName {
		resultsChan <- &KmsCheckIncludeResult{KeyId: ""}
		return
	}
	// additional request to describe key and get information about creation date, removal status
	details, err := svc.DescribeKey(&kms.DescribeKeyInput{KeyId: &key})
	if err != nil {
		resultsChan <- &KmsCheckIncludeResult{Error: err}
		return
	}
	metadata := details.KeyMetadata
	// evaluate only user keys
	if *metadata.KeyManager != kms.KeyManagerTypeCustomer {
		resultsChan <- &KmsCheckIncludeResult{KeyId: ""}
		return
	}
	// skip keys already scheduled for removal
	if metadata.DeletionDate != nil {
		resultsChan <- &KmsCheckIncludeResult{KeyId: ""}
		return
	}
	if metadata.PendingDeletionWindowInDays != nil {
		resultsChan <- &KmsCheckIncludeResult{KeyId: ""}
		return
	}
	referenceTime := *metadata.CreationDate
	if referenceTime.After(excludeAfter) {
		resultsChan <- &KmsCheckIncludeResult{KeyId: ""}
		return
	}
	// put key in channel to be considered for removal
	resultsChan <- &KmsCheckIncludeResult{KeyId: key}
}

func listKeyAliases(svc *kms.KMS, batchSize int) (map[string][]string, error) {
	// map key - KMS key id, value list of aliases
	aliases := map[string][]string{}
	var next *string

	for {
		list, err := svc.ListAliases(&kms.ListAliasesInput{
			Marker: next,
			Limit:  aws.Int64(int64(batchSize)),
		})
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		// collect key aliases to map
		for _, alias := range list.Aliases {
			key := alias.TargetKeyId
			if key == nil {
				continue
			}
			list, found := aliases[*key]
			if !found {
				list = make([]string, 0)
			}
			list = append(list, *alias.AliasName)
			aliases[*key] = list
		}

		if list.NextMarker == nil || len(*list.NextMarker) == 0 {
			break
		}
		next = list.NextMarker
	}
	return aliases, nil
}

func nukeAllCustomerManagedKmsKeys(session *session.Session, keyIds []*string, keyAliases map[string][]string) error {
	region := aws.StringValue(session.Config.Region)
	if len(keyIds) == 0 {
		logging.Logger.Debugf("No Customer Keys to nuke in region %s", region)
		return nil
	}

	// usage of go routines for parallel keys removal
	// https://docs.aws.amazon.com/sdk-for-go/api/service/kms/#KMS.ScheduleKeyDeletion
	logging.Logger.Debugf("Deleting Keys secrets in region %s", region)
	svc := kms.New(session)
	wg := new(sync.WaitGroup)
	wg.Add(len(keyIds))
	errChans := make([]chan error, len(keyIds))
	for i, secretID := range keyIds {
		errChans[i] = make(chan error, 1)
		go requestKeyDeletion(wg, errChans[i], svc, secretID)
	}
	wg.Wait()

	wgAlias := new(sync.WaitGroup)
	wgAlias.Add(len(keyAliases))
	for _, aliases := range keyAliases {
		go deleteAliases(wgAlias, svc, aliases)
	}
	wgAlias.Wait()

	// collect errors from each channel
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Logger.Debugf("[Failed] %s", err)
		}
	}
	return errors.WithStackTrace(allErrs.ErrorOrNil())
}

func deleteAliases(wg *sync.WaitGroup, svc *kms.KMS, aliases []string) {
	defer wg.Done()

	for _, aliasName := range aliases {
		input := &kms.DeleteAliasInput{AliasName: &aliasName}
		_, err := svc.DeleteAlias(input)

		if err != nil {
			logging.Logger.Errorf("[Failed] Failed deleting alias: %s", aliasName)
		} else {
			logging.Logger.Debugf("Deleted alias %s", aliasName)
		}
	}
}

func requestKeyDeletion(wg *sync.WaitGroup, errChan chan error, svc *kms.KMS, key *string) {
	defer wg.Done()
	input := &kms.ScheduleKeyDeletionInput{KeyId: key, PendingWindowInDays: aws.Int64(int64(kmsRemovalWindow))}
	_, err := svc.ScheduleKeyDeletion(input)

	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.StringValue(key),
		ResourceType: "Key Management Service (KMS) Key",
		Error:        err,
	}
	report.Record(e)

	errChan <- err
}
