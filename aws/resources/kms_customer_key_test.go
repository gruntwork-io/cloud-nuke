package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockKmsClient struct {
	ListKeysOutput            kms.ListKeysOutput
	ListAliasesOutput         kms.ListAliasesOutput
	DescribeKeyOutput         map[string]kms.DescribeKeyOutput
	ScheduleKeyDeletionOutput kms.ScheduleKeyDeletionOutput
}

func (m *mockKmsClient) ListKeys(ctx context.Context, params *kms.ListKeysInput, optFns ...func(*kms.Options)) (*kms.ListKeysOutput, error) {
	return &m.ListKeysOutput, nil
}

func (m *mockKmsClient) ListAliases(ctx context.Context, params *kms.ListAliasesInput, optFns ...func(*kms.Options)) (*kms.ListAliasesOutput, error) {
	return &m.ListAliasesOutput, nil
}

func (m *mockKmsClient) DescribeKey(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error) {
	output := m.DescribeKeyOutput[*params.KeyId]
	return &output, nil
}

func (m *mockKmsClient) ScheduleKeyDeletion(ctx context.Context, params *kms.ScheduleKeyDeletionInput, optFns ...func(*kms.Options)) (*kms.ScheduleKeyDeletionOutput, error) {
	return &m.ScheduleKeyDeletionOutput, nil
}

func TestListKmsCustomerKeys(t *testing.T) {
	t.Parallel()

	now := time.Now()
	key1, key2 := "key1", "key2"
	alias1, alias2 := "alias/key1", "alias/key2"

	mock := &mockKmsClient{
		ListKeysOutput: kms.ListKeysOutput{
			Keys: []types.KeyListEntry{
				{KeyId: aws.String(key1)},
				{KeyId: aws.String(key2)},
			},
		},
		ListAliasesOutput: kms.ListAliasesOutput{
			Aliases: []types.AliasListEntry{
				{AliasName: aws.String(alias1), TargetKeyId: aws.String(key1)},
				{AliasName: aws.String(alias2), TargetKeyId: aws.String(key2)},
			},
		},
		DescribeKeyOutput: map[string]kms.DescribeKeyOutput{
			key1: {KeyMetadata: &types.KeyMetadata{
				KeyId:        aws.String(key1),
				KeyManager:   types.KeyManagerTypeCustomer,
				CreationDate: aws.Time(now),
			}},
			key2: {KeyMetadata: &types.KeyMetadata{
				KeyId:        aws.String(key2),
				KeyManager:   types.KeyManagerTypeCustomer,
				CreationDate: aws.Time(now.Add(time.Hour)),
			}},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"noFilter": {
			configObj: config.ResourceType{},
			expected:  []string{key1, key2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(alias1)}},
				},
			},
			expected: []string{key2},
		},
		"nameInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(".*key1")}},
				},
			},
			expected: []string{key1},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{key1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listKmsCustomerKeys(context.Background(), mock, tc.configObj, false)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestListKmsCustomerKeys_IncludeUnaliasedKeys(t *testing.T) {
	t.Parallel()

	now := time.Now()
	keyWithAlias, keyWithoutAlias := "key-with-alias", "key-without-alias"

	mock := &mockKmsClient{
		ListKeysOutput: kms.ListKeysOutput{
			Keys: []types.KeyListEntry{
				{KeyId: aws.String(keyWithAlias)},
				{KeyId: aws.String(keyWithoutAlias)},
			},
		},
		ListAliasesOutput: kms.ListAliasesOutput{
			Aliases: []types.AliasListEntry{
				{AliasName: aws.String("alias/my-key"), TargetKeyId: aws.String(keyWithAlias)},
				// keyWithoutAlias has no alias
			},
		},
		DescribeKeyOutput: map[string]kms.DescribeKeyOutput{
			keyWithAlias: {KeyMetadata: &types.KeyMetadata{
				KeyId:        aws.String(keyWithAlias),
				KeyManager:   types.KeyManagerTypeCustomer,
				CreationDate: aws.Time(now),
			}},
			keyWithoutAlias: {KeyMetadata: &types.KeyMetadata{
				KeyId:        aws.String(keyWithoutAlias),
				KeyManager:   types.KeyManagerTypeCustomer,
				CreationDate: aws.Time(now),
			}},
		},
	}

	tests := map[string]struct {
		includeUnaliased bool
		expected         []string
	}{
		"excludeUnaliasedByDefault": {
			includeUnaliased: false,
			expected:         []string{keyWithAlias},
		},
		"includeUnaliasedWhenConfigured": {
			includeUnaliased: true,
			expected:         []string{keyWithAlias, keyWithoutAlias},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listKmsCustomerKeys(context.Background(), mock, config.ResourceType{}, tc.includeUnaliased)
			require.NoError(t, err)
			require.ElementsMatch(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestListKmsCustomerKeys_FiltersAwsManagedKeys(t *testing.T) {
	t.Parallel()

	now := time.Now()
	customerKey, awsKey := "customer-key", "aws-managed-key"

	mock := &mockKmsClient{
		ListKeysOutput: kms.ListKeysOutput{
			Keys: []types.KeyListEntry{
				{KeyId: aws.String(customerKey)},
				{KeyId: aws.String(awsKey)},
			},
		},
		ListAliasesOutput: kms.ListAliasesOutput{
			Aliases: []types.AliasListEntry{
				{AliasName: aws.String("alias/customer"), TargetKeyId: aws.String(customerKey)},
				{AliasName: aws.String("alias/aws/s3"), TargetKeyId: aws.String(awsKey)},
			},
		},
		DescribeKeyOutput: map[string]kms.DescribeKeyOutput{
			customerKey: {KeyMetadata: &types.KeyMetadata{
				KeyId:        aws.String(customerKey),
				KeyManager:   types.KeyManagerTypeCustomer,
				CreationDate: aws.Time(now),
			}},
			awsKey: {KeyMetadata: &types.KeyMetadata{
				KeyId:        aws.String(awsKey),
				KeyManager:   types.KeyManagerTypeAws, // AWS-managed key
				CreationDate: aws.Time(now),
			}},
		},
	}

	names, err := listKmsCustomerKeys(context.Background(), mock, config.ResourceType{}, false)
	require.NoError(t, err)
	require.Equal(t, []string{customerKey}, aws.ToStringSlice(names))
}

func TestListKmsCustomerKeys_SkipsPendingDeletion(t *testing.T) {
	t.Parallel()

	now := time.Now()
	activeKey, pendingKey := "active-key", "pending-deletion-key"
	deletionDate := now.Add(7 * 24 * time.Hour)

	mock := &mockKmsClient{
		ListKeysOutput: kms.ListKeysOutput{
			Keys: []types.KeyListEntry{
				{KeyId: aws.String(activeKey)},
				{KeyId: aws.String(pendingKey)},
			},
		},
		ListAliasesOutput: kms.ListAliasesOutput{
			Aliases: []types.AliasListEntry{
				{AliasName: aws.String("alias/active"), TargetKeyId: aws.String(activeKey)},
				{AliasName: aws.String("alias/pending"), TargetKeyId: aws.String(pendingKey)},
			},
		},
		DescribeKeyOutput: map[string]kms.DescribeKeyOutput{
			activeKey: {KeyMetadata: &types.KeyMetadata{
				KeyId:        aws.String(activeKey),
				KeyManager:   types.KeyManagerTypeCustomer,
				CreationDate: aws.Time(now),
			}},
			pendingKey: {KeyMetadata: &types.KeyMetadata{
				KeyId:                       aws.String(pendingKey),
				KeyManager:                  types.KeyManagerTypeCustomer,
				CreationDate:                aws.Time(now),
				DeletionDate:                &deletionDate,
				PendingDeletionWindowInDays: aws.Int32(7),
			}},
		},
	}

	names, err := listKmsCustomerKeys(context.Background(), mock, config.ResourceType{}, false)
	require.NoError(t, err)
	require.Equal(t, []string{activeKey}, aws.ToStringSlice(names))
}

func TestDeleteKmsCustomerKey(t *testing.T) {
	t.Parallel()

	mock := &mockKmsClient{
		ScheduleKeyDeletionOutput: kms.ScheduleKeyDeletionOutput{},
	}

	err := deleteKmsCustomerKey(context.Background(), mock, aws.String("test-key"))
	require.NoError(t, err)
}
