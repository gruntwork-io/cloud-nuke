package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// https://docs.aws.amazon.com/sdk-for-go/api/service/kms/#ScheduleKeyDeletionInput
// must be between 7 and 30, inclusive
const kmsRemovalWindow = 7

type KmsCustomerKeysAPI interface {
	ListKeys(ctx context.Context, params *kms.ListKeysInput, optFns ...func(*kms.Options)) (*kms.ListKeysOutput, error)
	ListAliases(ctx context.Context, params *kms.ListAliasesInput, optFns ...func(*kms.Options)) (*kms.ListAliasesOutput, error)
	DescribeKey(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error)
	DeleteAlias(ctx context.Context, params *kms.DeleteAliasInput, optFns ...func(*kms.Options)) (*kms.DeleteAliasOutput, error)
	ScheduleKeyDeletion(ctx context.Context, params *kms.ScheduleKeyDeletionInput, optFns ...func(*kms.Options)) (*kms.ScheduleKeyDeletionOutput, error)
}

type KmsCustomerKeys struct {
	BaseAwsResource
	Client     KmsCustomerKeysAPI
	Region     string
	KeyIds     []string
	KeyAliases map[string][]string
}

func (kck *KmsCustomerKeys) Init(cfg aws.Config) {
	kck.Client = kms.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (kck *KmsCustomerKeys) ResourceName() string {
	return "kmscustomerkeys"
}

// ResourceIdentifiers - The KMS Key IDs
func (kck *KmsCustomerKeys) ResourceIdentifiers() []string {
	return kck.KeyIds
}

// MaxBatchSize - Requests batch size
func (kck *KmsCustomerKeys) MaxBatchSize() int {
	return 49
}

func (kck *KmsCustomerKeys) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := kck.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	kck.KeyIds = aws.ToStringSlice(identifiers)
	return kck.KeyIds, nil
}

// Nuke - remove all customer managed keys
func (kck *KmsCustomerKeys) Nuke(keyIds []string) error {
	if err := kck.nukeAll(aws.StringSlice(keyIds)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
