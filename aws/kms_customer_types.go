package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
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

func (c KMSCustomerKeys) Nuke(session *session.Session, keyIds []string) error {
	if len(keyIds) == 0 {
		logging.Logger.Info("No Key IDs to nuke")
		return nil
	}
	svc := kms.New(session)
	var allErrs *multierror.Error
	for _, key := range keyIds {
		input := &kms.ScheduleKeyDeletionInput{KeyId: &key, PendingWindowInDays: aws.Int64(int64(kmsRemovalWindow))}
		_, err := svc.ScheduleKeyDeletion(input)
		if err != nil {
			allErrs = multierror.Append(allErrs, err)
		}
	}

	return errors.WithStackTrace(allErrs.ErrorOrNil())
}

