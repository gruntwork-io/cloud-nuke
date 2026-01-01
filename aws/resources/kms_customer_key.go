package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

// https://docs.aws.amazon.com/sdk-for-go/api/service/kms/#ScheduleKeyDeletionInput
// must be between 7 and 30, inclusive
const kmsRemovalWindow = 7

// Context key for passing IncludeUnaliasedKeys config
type kmsIncludeUnaliasedKeysKeyType struct{}

var kmsIncludeUnaliasedKeysKey = kmsIncludeUnaliasedKeysKeyType{}

// KmsCustomerKeysAPI defines the interface for KMS operations.
type KmsCustomerKeysAPI interface {
	ListKeys(ctx context.Context, params *kms.ListKeysInput, optFns ...func(*kms.Options)) (*kms.ListKeysOutput, error)
	ListAliases(ctx context.Context, params *kms.ListAliasesInput, optFns ...func(*kms.Options)) (*kms.ListAliasesOutput, error)
	DescribeKey(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error)
	DeleteAlias(ctx context.Context, params *kms.DeleteAliasInput, optFns ...func(*kms.Options)) (*kms.DeleteAliasOutput, error)
	ScheduleKeyDeletion(ctx context.Context, params *kms.ScheduleKeyDeletionInput, optFns ...func(*kms.Options)) (*kms.ScheduleKeyDeletionOutput, error)
}

// kmsState holds state that needs to be shared between list and nuke phases.
type kmsState struct {
	mu         sync.Mutex
	keyAliases map[string][]string
}

// globalKMSState is the global state for KMS operations.
// This is needed because the generic pattern separates list and nuke into different functions.
var globalKMSState = &kmsState{
	keyAliases: make(map[string][]string),
}

// NewKmsCustomerKeys creates a new KMS Customer Keys resource using the generic resource pattern.
func NewKmsCustomerKeys() AwsResource {
	return NewAwsResource(&resource.Resource[KmsCustomerKeysAPI]{
		ResourceTypeName: "kmscustomerkeys",
		BatchSize:        49,
		InitClient: func(r *resource.Resource[KmsCustomerKeysAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for KMS client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = kms.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.KMSCustomerKeys.ResourceType
		},
		Lister: listKmsCustomerKeys,
		Nuker:  deleteKmsCustomerKeys,
	})
}

// KmsCheckIncludeResult - structure used for results of evaluation
type KmsCheckIncludeResult struct {
	KeyId string
	Error error
}

// listKmsCustomerKeys retrieves all KMS customer keys that match the config filters.
func listKmsCustomerKeys(ctx context.Context, client KmsCustomerKeysAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	// Try to get IncludeUnaliasedKeys from context (set by the outer caller if needed)
	includeUnaliasedKeys := false
	if val, ok := ctx.Value(kmsIncludeUnaliasedKeysKey).(bool); ok {
		includeUnaliasedKeys = val
	}

	// Collect all keys in the account
	var keys []string
	listKeysPaginator := kms.NewListKeysPaginator(client, &kms.ListKeysInput{})
	for listKeysPaginator.HasMorePages() {
		page, err := listKeysPaginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, key := range page.Keys {
			keys = append(keys, *key.KeyId)
		}
	}

	// Collect key to alias mapping
	keyAliases := map[string][]string{}
	listAliasesPaginator := kms.NewListAliasesPaginator(client, &kms.ListAliasesInput{})
	for listAliasesPaginator.HasMorePages() {
		page, err := listAliasesPaginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

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
	}

	// checking in parallel if keys can be considered for removal
	var wg sync.WaitGroup
	wg.Add(len(keys))
	resultsChan := make([]chan *KmsCheckIncludeResult, len(keys))
	id := 0

	for _, keyId := range keys {
		resultsChan[id] = make(chan *KmsCheckIncludeResult, 1)

		// If the keyId isn't found in the map, this returns an empty array
		aliasesForKey := keyAliases[keyId]

		go shouldIncludeKmsKey(ctx, client, &wg, resultsChan[id], keyId, aliasesForKey, cfg, includeUnaliasedKeys)
		id++
	}
	wg.Wait()

	var kmsIds []*string
	aliases := map[string][]string{}

	for _, channel := range resultsChan {
		result := <-channel
		if result.Error != nil {
			logging.Debugf("Can't read KMS key %s", result.Error)

			continue
		}
		if result.KeyId != "" {
			aliases[result.KeyId] = keyAliases[result.KeyId]
			kmsIds = append(kmsIds, &result.KeyId)
		}
	}

	// Store aliases in global state for use during nuke phase
	globalKMSState.mu.Lock()
	globalKMSState.keyAliases = aliases
	globalKMSState.mu.Unlock()

	return kmsIds, nil
}

func shouldIncludeKmsKey(
	ctx context.Context,
	client KmsCustomerKeysAPI,
	wg *sync.WaitGroup,
	resultsChan chan *KmsCheckIncludeResult,
	key string,
	aliases []string,
	cfg config.ResourceType,
	includeUnaliasedKeys bool) {
	defer wg.Done()

	includedByName := false
	// verify if key aliases matches configurations
	for _, alias := range aliases {
		if config.ShouldInclude(&alias, cfg.IncludeRule.NamesRegExp,
			cfg.ExcludeRule.NamesRegExp) {
			includedByName = true
		}
	}

	// Only delete keys without aliases if the user explicitly says so
	if len(aliases) == 0 {
		if !includeUnaliasedKeys {
			resultsChan <- &KmsCheckIncludeResult{KeyId: ""}
			return
		} else {
			// Set this to true so keys w/o aliases don't bail out
			// On the !includedByName check
			includedByName = true
		}
	}

	if !includedByName {
		resultsChan <- &KmsCheckIncludeResult{KeyId: ""}
		return
	}
	// additional request to describe key and get information about creation date, removal status
	details, err := client.DescribeKey(ctx, &kms.DescribeKeyInput{KeyId: &key})
	if err != nil {
		resultsChan <- &KmsCheckIncludeResult{Error: err}
		return
	}
	metadata := details.KeyMetadata
	// evaluate only user keys
	if metadata.KeyManager != types.KeyManagerTypeCustomer {
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
	// Check time-based filtering (name filtering was already done above)
	referenceTime := metadata.CreationDate
	if !cfg.ShouldIncludeBasedOnTime(*referenceTime) {
		resultsChan <- &KmsCheckIncludeResult{KeyId: ""}
		return
	}
	// put key in channel to be considered for removal
	resultsChan <- &KmsCheckIncludeResult{KeyId: key}
}

// deleteKmsCustomerKeys deletes KMS customer keys.
func deleteKmsCustomerKeys(ctx context.Context, client KmsCustomerKeysAPI, scope resource.Scope, resourceType string, keyIds []*string) error {
	if len(keyIds) == 0 {
		logging.Debugf("No Customer Keys to nuke in region %s", scope.Region)
		return nil
	}

	// Get aliases from global state
	globalKMSState.mu.Lock()
	keyAliases := globalKMSState.keyAliases
	globalKMSState.mu.Unlock()

	// usage of go routines for parallel keys removal
	logging.Debugf("Deleting Keys secrets in region %s", scope.Region)
	wg := new(sync.WaitGroup)
	wg.Add(len(keyIds))
	errChans := make([]chan error, len(keyIds))
	for i, secretID := range keyIds {
		errChans[i] = make(chan error, 1)
		go requestKmsKeyDeletion(ctx, client, wg, errChans[i], secretID)
	}
	wg.Wait()

	wgAlias := new(sync.WaitGroup)
	wgAlias.Add(len(keyAliases))
	for _, aliases := range keyAliases {
		go deleteKmsAliases(ctx, client, wgAlias, aliases)
	}
	wgAlias.Wait()

	// collect errors from each channel
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Debugf("[Failed] %s", err)
		}
	}
	return errors.WithStackTrace(allErrs.ErrorOrNil())
}

func deleteKmsAliases(ctx context.Context, client KmsCustomerKeysAPI, wg *sync.WaitGroup, aliases []string) {
	defer wg.Done()

	for _, aliasName := range aliases {
		input := &kms.DeleteAliasInput{AliasName: &aliasName}
		_, err := client.DeleteAlias(ctx, input)

		if err != nil {
			logging.Errorf("[Failed] Failed deleting alias: %s", aliasName)
		} else {
			logging.Debugf("Deleted alias %s", aliasName)
		}
	}
}

func requestKmsKeyDeletion(ctx context.Context, client KmsCustomerKeysAPI, wg *sync.WaitGroup, errChan chan error, key *string) {
	defer wg.Done()
	input := &kms.ScheduleKeyDeletionInput{KeyId: key, PendingWindowInDays: aws.Int32(int32(kmsRemovalWindow))}
	_, err := client.ScheduleKeyDeletion(ctx, input)

	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.ToString(key),
		ResourceType: "Key Management Service (KMS) Key",
		Error:        err,
	}
	report.Record(e)

	errChan <- err
}
