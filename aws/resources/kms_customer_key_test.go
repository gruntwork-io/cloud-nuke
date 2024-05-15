package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedKmsCustomerKeys struct {
	kmsiface.KMSAPI
	ListKeysPagesOutput       kms.ListKeysOutput
	ListAliasesPagesOutput    kms.ListAliasesOutput
	DescribeKeyOutput         map[string]kms.DescribeKeyOutput
	ScheduleKeyDeletionOutput kms.ScheduleKeyDeletionOutput
	DeleteAliasOutput         kms.DeleteAliasOutput
}

func (m mockedKmsCustomerKeys) ListKeysPagesWithContext(_ aws.Context, input *kms.ListKeysInput, fn func(*kms.ListKeysOutput, bool) bool, _ ...request.Option) error {
	fn(&m.ListKeysPagesOutput, true)
	return nil
}

func (m mockedKmsCustomerKeys) ListAliasesPagesWithContext(_ aws.Context, input *kms.ListAliasesInput, fn func(*kms.ListAliasesOutput, bool) bool, _ ...request.Option) error {
	fn(&m.ListAliasesPagesOutput, true)
	return nil
}

func (m mockedKmsCustomerKeys) DescribeKeyWithContext(_ aws.Context, input *kms.DescribeKeyInput, _ ...request.Option) (*kms.DescribeKeyOutput, error) {
	id := input.KeyId
	output := m.DescribeKeyOutput[*id]

	return &output, nil
}

func (m mockedKmsCustomerKeys) ScheduleKeyDeletionWithContext(_ aws.Context, input *kms.ScheduleKeyDeletionInput, _ ...request.Option) (*kms.ScheduleKeyDeletionOutput, error) {
	return &m.ScheduleKeyDeletionOutput, nil
}

func (m mockedKmsCustomerKeys) DeleteAliasWithContext(_ aws.Context, input *kms.DeleteAliasInput, _ ...request.Option) (*kms.DeleteAliasOutput, error) {
	return &m.DeleteAliasOutput, nil
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
			ListKeysPagesOutput: kms.ListKeysOutput{
				Keys: []*kms.KeyListEntry{
					{
						KeyId: aws.String(key1),
					},
					{
						KeyId: aws.String(key2),
					},
				},
			},
			ListAliasesPagesOutput: kms.ListAliasesOutput{
				Aliases: []*kms.AliasListEntry{
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
					KeyMetadata: &kms.KeyMetadata{
						KeyId:        aws.String(key1),
						KeyManager:   aws.String(kms.KeyManagerTypeCustomer),
						CreationDate: aws.Time(now),
					},
				},
				key2: {
					KeyMetadata: &kms.KeyMetadata{
						KeyId:        aws.String(key2),
						KeyManager:   aws.String(kms.KeyManagerTypeCustomer),
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
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
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
