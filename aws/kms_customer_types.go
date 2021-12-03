package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// https://docs.aws.amazon.com/sdk-for-go/api/service/kms/#ScheduleKeyDeletionInput
// must be between 7 and 30, inclusive
const kmsRemovalWindow = 7

type KMSCustomerKeys struct {
	KeyIds []string
}

// ResourceName - the simple name of the aws resource
func (c KMSCustomerKeys) ResourceName() string {
	return "kms-customer"
}

// ResourceIdentifiers - The IAM UserNames
func (c KMSCustomerKeys) ResourceIdentifiers() []string {
	return c.KeyIds
}

// MaxBatchSize - Requests batch size
func (r KMSCustomerKeys) MaxBatchSize() int {
	return 100
}

// Nuke - remove all customer managed keys
func (c KMSCustomerKeys) Nuke(session *session.Session, keyIds []string) error {
	if err := nukeAllCustomerManagedKmsKeys(session, awsgo.StringSlice(keyIds)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

