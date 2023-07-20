package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// https://docs.aws.amazon.com/sdk-for-go/api/service/kms/#ScheduleKeyDeletionInput
// must be between 7 and 30, inclusive
const kmsRemovalWindow = 7

type KmsCustomerKey struct {
	Client     kmsiface.KMSAPI
	Region     string
	KeyIds     []string
	KeyAliases map[string][]string
}

// ResourceName - the simple name of the aws resource
func (c KmsCustomerKey) ResourceName() string {
	return "kms-customer-keys"
}

// ResourceIdentifiers - The KMS Key IDs
func (c KmsCustomerKey) ResourceIdentifiers() []string {
	return c.KeyIds
}

// MaxBatchSize - Requests batch size
func (r KmsCustomerKey) MaxBatchSize() int {
	return 49
}

// Nuke - remove all customer managed keys
func (c KmsCustomerKey) Nuke(session *session.Session, keyIds []string) error {
	if err := nukeAllCustomerManagedKmsKeys(session, awsgo.StringSlice(keyIds), c.KeyAliases); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
