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

type mockedKmsCustomerKeys struct {
	KmsCustomerKeysAPI
	ListKeysOutput            kms.ListKeysOutput
	ListAliasesOutput         kms.ListAliasesOutput
	DescribeKeyOutput         map[string]kms.DescribeKeyOutput
	DeleteAliasOutput         kms.DeleteAliasOutput
	ScheduleKeyDeletionOutput kms.ScheduleKeyDeletionOutput
}

func (m mockedKmsCustomerKeys) ListKeys(ctx context.Context, params *kms.ListKeysInput, optFns ...func(*kms.Options)) (*kms.ListKeysOutput, error) {
	return &m.ListKeysOutput, nil
}

func (m mockedKmsCustomerKeys) ListAliases(ctx context.Context, params *kms.ListAliasesInput, optFns ...func(*kms.Options)) (*kms.ListAliasesOutput, error) {
	return &m.ListAliasesOutput, nil
}

func (m mockedKmsCustomerKeys) DescribeKey(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error) {
	id := params.KeyId
	output := m.DescribeKeyOutput[*id]
	return &output, nil
}

func (m mockedKmsCustomerKeys) DeleteAlias(ctx context.Context, params *kms.DeleteAliasInput, optFns ...func(*kms.Options)) (*kms.DeleteAliasOutput, error) {
	return &m.DeleteAliasOutput, nil
}

func (m mockedKmsCustomerKeys) ScheduleKeyDeletion(ctx context.Context, params *kms.ScheduleKeyDeletionInput, optFns ...func(*kms.Options)) (*kms.ScheduleKeyDeletionOutput, error) {
	return &m.ScheduleKeyDeletionOutput, nil
}

func TestKMS_GetAll(t *testing.T) {
	t.Parallel()

	key1 := "key1"
	key2 := "key2"
	alias1 := "alias/key1"
	alias2 := "alias/key2"
	now := time.Now()
	kck := KmsCustomerKeys{
		Client: mockedKmsCustomerKeys{
			ListKeysOutput: kms.ListKeysOutput{
				Keys: []types.KeyListEntry{
					{
						KeyId: aws.String(key1),
					},
					{
						KeyId: aws.String(key2),
					},
				},
			},
			ListAliasesOutput: kms.ListAliasesOutput{
				Aliases: []types.AliasListEntry{
					{
						AliasName:   aws.String(alias1),
						TargetKeyId: aws.String(key1),
					},
					{
						AliasName:   aws.String(alias2),
						TargetKeyId: aws.String(key2),
					},
				},
			},
			DescribeKeyOutput: map[string]kms.DescribeKeyOutput{
				key1: {
					KeyMetadata: &types.KeyMetadata{
						KeyId:        aws.String(key1),
						KeyManager:   types.KeyManagerTypeCustomer,
						CreationDate: aws.Time(now),
					},
				},
				key2: {
					KeyMetadata: &types.KeyMetadata{
						KeyId:        aws.String(key2),
						KeyManager:   types.KeyManagerTypeCustomer,
						CreationDate: aws.Time(now.Add(1)),
					},
				},
			},
		},
	}

	tests := map[string]struct {
		configObj config.KMSCustomerKeyResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.KMSCustomerKeyResourceType{},
			expected:  []string{key1, key2},
		},
		"nameExclusionFilter": {
			configObj: config.KMSCustomerKeyResourceType{
				ResourceType: config.ResourceType{
					ExcludeRule: config.FilterRule{
						NamesRegExp: []config.Expression{{
							RE: *regexp.MustCompile(alias1),
						}}},
				},
			},
			expected: []string{key2},
		},
		"nameInclusionFilter": {
			configObj: config.KMSCustomerKeyResourceType{
				ResourceType: config.ResourceType{
					IncludeRule: config.FilterRule{
						NamesRegExp: []config.Expression{{
							RE: *regexp.MustCompile(".*key1"),
						}}},
				},
			},
			expected: []string{key1},
		},
		"timeAfterExclusionFilter": {
			configObj: config.KMSCustomerKeyResourceType{
				ResourceType: config.ResourceType{
					ExcludeRule: config.FilterRule{
						TimeAfter: aws.Time(now),
					}},
			},
			expected: []string{key1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := kck.getAll(context.Background(), config.Config{
				KMSCustomerKeys: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestKMS_GetAll_IncludeUnaliased(t *testing.T) {
	t.Parallel()

	key1 := "key-with-alias"
	key2 := "key-without-alias"
	alias1 := "alias/my-key"
	now := time.Now()
	kck := KmsCustomerKeys{
		Client: mockedKmsCustomerKeys{
			ListKeysOutput: kms.ListKeysOutput{
				Keys: []types.KeyListEntry{
					{KeyId: aws.String(key1)},
					{KeyId: aws.String(key2)},
				},
			},
			ListAliasesOutput: kms.ListAliasesOutput{
				Aliases: []types.AliasListEntry{
					{
						AliasName:   aws.String(alias1),
						TargetKeyId: aws.String(key1),
					},
					// key2 has no alias
				},
			},
			DescribeKeyOutput: map[string]kms.DescribeKeyOutput{
				key1: {
					KeyMetadata: &types.KeyMetadata{
						KeyId:        aws.String(key1),
						KeyManager:   types.KeyManagerTypeCustomer,
						CreationDate: aws.Time(now),
					},
				},
				key2: {
					KeyMetadata: &types.KeyMetadata{
						KeyId:        aws.String(key2),
						KeyManager:   types.KeyManagerTypeCustomer,
						CreationDate: aws.Time(now),
					},
				},
			},
		},
	}

	tests := map[string]struct {
		configObj config.KMSCustomerKeyResourceType
		expected  []string
	}{
		"excludeUnaliasedByDefault": {
			configObj: config.KMSCustomerKeyResourceType{
				IncludeUnaliasedKeys: false,
			},
			expected: []string{key1}, // only key with alias
		},
		"includeUnaliasedWhenConfigured": {
			configObj: config.KMSCustomerKeyResourceType{
				IncludeUnaliasedKeys: true,
			},
			expected: []string{key1, key2}, // both keys
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := kck.getAll(context.Background(), config.Config{
				KMSCustomerKeys: tc.configObj,
			})
			require.NoError(t, err)
			require.ElementsMatch(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestKMS_NukeAll(t *testing.T) {
	t.Parallel()

	kck := KmsCustomerKeys{
		Client: mockedKmsCustomerKeys{
			DeleteAliasOutput:         kms.DeleteAliasOutput{},
			ScheduleKeyDeletionOutput: kms.ScheduleKeyDeletionOutput{},
		},
	}

	err := kck.nukeAll([]*string{aws.String("key1"), aws.String("key2")})
	require.NoError(t, err)
}
