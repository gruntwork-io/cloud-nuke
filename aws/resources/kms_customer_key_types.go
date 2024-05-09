package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// https://docs.aws.amazon.com/sdk-for-go/api/service/kms/#ScheduleKeyDeletionInput
// must be between 7 and 30, inclusive
const kmsRemovalWindow = 7

type KmsCustomerKeys struct {
	BaseAwsResource
	Client     kmsiface.KMSAPI
	Region     string
	KeyIds     []string
	KeyAliases map[string][]string
}

func (kck *KmsCustomerKeys) Init(session *session.Session) {
	kck.Client = kms.New(session)
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

// func (kck *KmsCustomerKeys) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
// 	return configObj.KMSCustomerKeys
// }

func (kck *KmsCustomerKeys) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := kck.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	kck.KeyIds = awsgo.StringValueSlice(identifiers)
	return kck.KeyIds, nil
}

// Nuke - remove all customer managed keys
func (kck *KmsCustomerKeys) Nuke(keyIds []string) error {
	if err := kck.nukeAll(awsgo.StringSlice(keyIds)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
