package resources

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"sync"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/report"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

func (kck *KmsCustomerKeys) getAll(configObj config.Config) ([]*string, error) {
	// Collect all keys in the account
	var keys []string
	err := kck.Client.ListKeysPages(&kms.ListKeysInput{}, func(page *kms.ListKeysOutput, lastPage bool) bool {
		for _, key := range page.Keys {
			keys = append(keys, *key.KeyId)
		}

		return !lastPage
	})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	// Collect key to alias mapping
	keyAliases := map[string][]string{}
	err = kck.Client.ListAliasesPages(&kms.ListAliasesInput{}, func(page *kms.ListAliasesOutput, lastPage bool) bool {
		for _, alias := range page.Aliases {
			key := alias.TargetKeyId
			if key == nil {
				continue
			}

			list, found := keyAliases[*key]
			if !found {
				list = make([]string, 0)
			}

			list = append(list, *alias.AliasName)
			keyAliases[*key] = list
		}

		return !lastPage
	})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	// checking in parallel if keys can be considered for removal
	var wg sync.WaitGroup
	wg.Add(len(keys))
	resultsChan := make([]chan *KmsCheckIncludeResult, len(keys))
	id := 0

	for _, keyId := range keys {
		resultsChan[id] = make(chan *KmsCheckIncludeResult, 1)

		if err != nil {
			logging.Logger.Debugf("Can't read KMS key %s", err.Error())
			continue
		}

		// If the keyId isn't found in the map, this returns an empty array
		aliasesForKey := keyAliases[keyId]

		go kck.shouldInclude(&wg, resultsChan[id], keyId, aliasesForKey, configObj)
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

	kck.KeyAliases = aliases
	return kmsIds, nil
}

// KmsCheckIncludeResult - structure used results of evaluation: not null KeyId - key should be included
type KmsCheckIncludeResult struct {
	KeyId string
	Error error
}

func (kck *KmsCustomerKeys) shouldInclude(
	wg *sync.WaitGroup,
	resultsChan chan *KmsCheckIncludeResult,
	key string,
	aliases []string,
	configObj config.Config) {
	defer wg.Done()

	includedByName := false
	// verify if key aliases matches configurations
	for _, alias := range aliases {
		if config.ShouldInclude(alias, configObj.KMSCustomerKeys.IncludeRule.NamesRegExp,
			configObj.KMSCustomerKeys.ExcludeRule.NamesRegExp) {
			includedByName = true
		}
	}

	// Only delete keys without aliases if the user explicitly says so
	if len(aliases) == 0 {
		if !configObj.KMSCustomerKeys.IncludeUnaliasedKeys {
			resultsChan <- &KmsCheckIncludeResult{KeyId: ""}
			return
		} else {
			// Set this to true so keys w/o aliases dont bail out
			// On the !includedByName check
			includedByName = true
		}
	}

	if !includedByName {
		resultsChan <- &KmsCheckIncludeResult{KeyId: ""}
		return
	}
	// additional request to describe key and get information about creation date, removal status
	details, err := kck.Client.DescribeKey(&kms.DescribeKeyInput{KeyId: &key})
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
	referenceTime := metadata.CreationDate
	if !configObj.KMSCustomerKeys.ShouldInclude(config.ResourceValue{Time: referenceTime}) {
		resultsChan <- &KmsCheckIncludeResult{KeyId: ""}
		return
	}
	// put key in channel to be considered for removal
	resultsChan <- &KmsCheckIncludeResult{KeyId: key}
}

func (kck *KmsCustomerKeys) nukeAll(keyIds []*string) error {
	if len(keyIds) == 0 {
		logging.Logger.Debugf("No Customer Keys to nuke in region %s", kck.Region)
		return nil
	}

	// usage of go routines for parallel keys removal
	// https://docs.aws.amazon.com/sdk-for-go/api/service/kms/#KMS.ScheduleKeyDeletion
	logging.Logger.Debugf("Deleting Keys secrets in region %s", kck.Region)
	wg := new(sync.WaitGroup)
	wg.Add(len(keyIds))
	errChans := make([]chan error, len(keyIds))
	for i, secretID := range keyIds {
		errChans[i] = make(chan error, 1)
		go kck.requestKeyDeletion(wg, errChans[i], secretID)
	}
	wg.Wait()

	wgAlias := new(sync.WaitGroup)
	wgAlias.Add(len(kck.KeyAliases))
	for _, aliases := range kck.KeyAliases {
		go kck.deleteAliases(wgAlias, aliases)
	}
	wgAlias.Wait()

	// collect errors from each channel
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking KMS Key",
			}, map[string]interface{}{
				"region": kck.Region,
			})
		}
	}
	return errors.WithStackTrace(allErrs.ErrorOrNil())
}

func (kck *KmsCustomerKeys) deleteAliases(wg *sync.WaitGroup, aliases []string) {
	defer wg.Done()

	for _, aliasName := range aliases {
		input := &kms.DeleteAliasInput{AliasName: &aliasName}
		_, err := kck.Client.DeleteAlias(input)

		if err != nil {
			logging.Logger.Errorf("[Failed] Failed deleting alias: %s", aliasName)
		} else {
			logging.Logger.Debugf("Deleted alias %s", aliasName)
		}
	}
}

func (kck *KmsCustomerKeys) requestKeyDeletion(wg *sync.WaitGroup, errChan chan error, key *string) {
	defer wg.Done()
	input := &kms.ScheduleKeyDeletionInput{KeyId: key, PendingWindowInDays: aws.Int64(int64(kmsRemovalWindow))}
	_, err := kck.Client.ScheduleKeyDeletion(input)

	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.StringValue(key),
		ResourceType: "Key Management Service (KMS) Key",
		Error:        err,
	}
	report.Record(e)

	errChan <- err
}
