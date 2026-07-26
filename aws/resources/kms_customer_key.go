package resources

import (
	"context"
	"strings"

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
	ListResourceTags(ctx context.Context, params *kms.ListResourceTagsInput, optFns ...func(*kms.Options)) (*kms.ListResourceTagsOutput, error)
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
		ResourceTypeName: "kms-customer-key",
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

// kmsAliasPrefix is the prefix AWS returns on every alias from ListAliases.
const kmsAliasPrefix = "alias/"

// aliasNameCandidates returns the strings that name filters are matched against for a
// given set of aliases. AWS reports aliases as "alias/my-key", but the "alias/" prefix is
// an API artifact rather than part of the name users think in, so both the full alias and
// the bare name are considered. A pattern matching either form applies to the key.
func aliasNameCandidates(aliases []string) []string {
	candidates := make([]string, 0, len(aliases)*2)
	for _, alias := range aliases {
		candidates = append(candidates, alias)
		if bare := strings.TrimPrefix(alias, kmsAliasPrefix); bare != alias {
			candidates = append(candidates, bare)
		}
	}
	return candidates
}

// matchesNameFilters reports whether a key passes the configured name filters.
//
// An exclude rule matching any form of any alias protects the whole key, so that a key
// carrying several aliases cannot be deleted just because one of its other aliases was
// not excluded. When include rules are present, at least one form must match.
func matchesNameFilters(aliases []string, cfg config.ResourceType) bool {
	candidates := aliasNameCandidates(aliases)

	for _, name := range candidates {
		if !config.ShouldInclude(&name, nil, cfg.ExcludeRule.NamesRegExp) {
			return false
		}
	}

	if len(cfg.IncludeRule.NamesRegExp) == 0 {
		return true
	}

	for _, name := range candidates {
		if config.ShouldInclude(&name, cfg.IncludeRule.NamesRegExp, nil) {
			return true
		}
	}
	return false
}

// shouldIncludeKey determines if a key should be included for deletion.
func shouldIncludeKey(ctx context.Context, client KmsCustomerKeysAPI, keyId string, aliases []string, cfg config.ResourceType, includeUnaliasedKeys bool) (bool, error) {
	// Skip keys without aliases unless explicitly configured to include them. Unaliased
	// keys have no name to match, so they bypass name filtering entirely.
	if len(aliases) == 0 {
		if !includeUnaliasedKeys {
			return false, nil
		}
	} else if !matchesNameFilters(aliases, cfg) {
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

	// Check tag-based filtering
	tags, err := getKmsKeyTags(ctx, client, keyId)
	if err != nil {
		logging.Debugf("Error getting tags for KMS key %s: %v", keyId, err)
		// If we can't get tags, pass empty map so tag filters correctly exclude this resource
		tags = map[string]string{}
	}
	if !cfg.ShouldIncludeBasedOnTag(tags) {
		return false, nil
	}

	return true, nil
}

// getKmsKeyTags retrieves all tags for a KMS key as a map.
func getKmsKeyTags(ctx context.Context, client KmsCustomerKeysAPI, keyId string) (map[string]string, error) {
	output, err := client.ListResourceTags(ctx, &kms.ListResourceTagsInput{
		KeyId: &keyId,
	})
	if err != nil {
		return nil, err
	}

	tagMap := make(map[string]string)
	for _, tag := range output.Tags {
		if tag.TagKey != nil && tag.TagValue != nil {
			tagMap[*tag.TagKey] = *tag.TagValue
		}
	}
	return tagMap, nil
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
