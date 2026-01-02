package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// kmsRemovalWindow is the number of days before a scheduled key is permanently deleted.
// https://docs.aws.amazon.com/kms/latest/APIReference/API_ScheduleKeyDeletion.html
// Must be between 7 and 30 days, inclusive.
const kmsRemovalWindow = 7

// KmsCustomerKeysAPI defines the interface for KMS operations.
type KmsCustomerKeysAPI interface {
	ListKeys(ctx context.Context, params *kms.ListKeysInput, optFns ...func(*kms.Options)) (*kms.ListKeysOutput, error)
	ListAliases(ctx context.Context, params *kms.ListAliasesInput, optFns ...func(*kms.Options)) (*kms.ListAliasesOutput, error)
	DescribeKey(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error)
	ScheduleKeyDeletion(ctx context.Context, params *kms.ScheduleKeyDeletionInput, optFns ...func(*kms.Options)) (*kms.ScheduleKeyDeletionOutput, error)
}

// kmsCustomerKeysResource holds additional config that the lister needs
type kmsCustomerKeysResource struct {
	includeUnaliasedKeys bool
}

// NewKmsCustomerKeys creates a new KMS Customer Keys resource using the generic resource pattern.
func NewKmsCustomerKeys() AwsResource {
	kmsResource := &kmsCustomerKeysResource{}

	return NewAwsResource(&resource.Resource[KmsCustomerKeysAPI]{
		ResourceTypeName: "kmscustomerkeys",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[KmsCustomerKeysAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = kms.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			// Capture the IncludeUnaliasedKeys setting for use in the lister
			kmsResource.includeUnaliasedKeys = c.KMSCustomerKeys.IncludeUnaliasedKeys
			return c.KMSCustomerKeys.ResourceType
		},
		Lister: func(ctx context.Context, client KmsCustomerKeysAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
			return listKmsCustomerKeys(ctx, client, cfg, kmsResource.includeUnaliasedKeys)
		},
		Nuker: resource.SimpleBatchDeleter(deleteKmsCustomerKey),
	})
}

// listKmsCustomerKeys retrieves all KMS customer keys that match the config filters.
func listKmsCustomerKeys(ctx context.Context, client KmsCustomerKeysAPI, cfg config.ResourceType, includeUnaliasedKeys bool) ([]*string, error) {
	// Collect all keys using pagination
	keys, err := getAllKeys(ctx, client)
	if err != nil {
		return nil, err
	}

	// Build key to aliases mapping using pagination
	keyAliases, err := getKeyAliasesMap(ctx, client)
	if err != nil {
		return nil, err
	}

	// Filter keys based on configuration
	var result []*string
	for _, keyId := range keys {
		shouldInclude, err := shouldIncludeKey(ctx, client, keyId, keyAliases[keyId], cfg, includeUnaliasedKeys)
		if err != nil {
			logging.Debugf("Error checking KMS key %s: %v", keyId, err)
			continue
		}
		if shouldInclude {
			id := keyId // Create a copy for the pointer
			result = append(result, &id)
		}
	}

	return result, nil
}

// getAllKeys retrieves all KMS key IDs using pagination.
func getAllKeys(ctx context.Context, client KmsCustomerKeysAPI) ([]string, error) {
	var keys []string
	paginator := kms.NewListKeysPaginator(client, &kms.ListKeysInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, key := range page.Keys {
			if key.KeyId != nil {
				keys = append(keys, *key.KeyId)
			}
		}
	}

	return keys, nil
}

// getKeyAliasesMap builds a mapping from key ID to its aliases using pagination.
func getKeyAliasesMap(ctx context.Context, client KmsCustomerKeysAPI) (map[string][]string, error) {
	keyAliases := make(map[string][]string)
	paginator := kms.NewListAliasesPaginator(client, &kms.ListAliasesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, alias := range page.Aliases {
			if alias.TargetKeyId == nil || alias.AliasName == nil {
				continue
			}
			keyAliases[*alias.TargetKeyId] = append(keyAliases[*alias.TargetKeyId], *alias.AliasName)
		}
	}

	return keyAliases, nil
}

// shouldIncludeKey determines if a key should be included for deletion.
func shouldIncludeKey(ctx context.Context, client KmsCustomerKeysAPI, keyId string, aliases []string, cfg config.ResourceType, includeUnaliasedKeys bool) (bool, error) {
	// Skip keys without aliases unless explicitly configured to include them
	if len(aliases) == 0 && !includeUnaliasedKeys {
		return false, nil
	}

	// Check if any alias matches the name filter
	matchedByName := len(aliases) == 0 && includeUnaliasedKeys // Unaliased keys pass if configured
	for _, alias := range aliases {
		if config.ShouldInclude(&alias, cfg.IncludeRule.NamesRegExp, cfg.ExcludeRule.NamesRegExp) {
			matchedByName = true
			break
		}
	}
	if !matchedByName {
		return false, nil
	}

	// Get key metadata to check additional filters
	details, err := client.DescribeKey(ctx, &kms.DescribeKeyInput{KeyId: &keyId})
	if err != nil {
		return false, err
	}

	metadata := details.KeyMetadata
	if metadata == nil {
		return false, nil
	}

	// Only include customer-managed keys (not AWS-managed)
	if metadata.KeyManager != types.KeyManagerTypeCustomer {
		return false, nil
	}

	// Skip keys already scheduled for deletion
	if metadata.DeletionDate != nil || metadata.PendingDeletionWindowInDays != nil {
		return false, nil
	}

	// Check time-based filtering
	if metadata.CreationDate != nil && !cfg.ShouldIncludeBasedOnTime(*metadata.CreationDate) {
		return false, nil
	}

	return true, nil
}

// deleteKmsCustomerKey schedules a single KMS customer key for deletion.
// AWS automatically deletes all aliases associated with a key when the key is deleted.
func deleteKmsCustomerKey(ctx context.Context, client KmsCustomerKeysAPI, keyId *string) error {
	_, err := client.ScheduleKeyDeletion(ctx, &kms.ScheduleKeyDeletionInput{
		KeyId:               keyId,
		PendingWindowInDays: aws.Int32(kmsRemovalWindow),
	})
	return err
}
