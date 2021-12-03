package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
	"time"
)



func getAllKmsUserKeys(session *session.Session, batchSize int, excludeAfter time.Time) ([]*string, error) {
	svc := kms.New(session)
	var kmsIds []*string
	input := &kms.ListKeysInput{
		Limit: aws.Int64(int64(batchSize)),
	}
	listPage := 1
	err := svc.ListKeysPages(input, func(page *kms.ListKeysOutput, lastPage bool) bool {
		logging.Logger.Debugf("Loading User Key %d", listPage)

		for _, key := range page.Keys {
			include, err := shouldIncludeKmsUserKey(svc, key, excludeAfter)
			if err != nil {
				logging.Logger.Errorf("Occured error during checking key %v", err)
				return false
			}
			if  include{
				logging.Logger.Debugf("key.KeyArn %v", key.KeyArn)
				kmsIds = append(kmsIds, key.KeyId)
			}
		}
		listPage++
		return true
	})

	return kmsIds, errors.WithStackTrace(err)
}

func shouldIncludeKmsUserKey(svc *kms.KMS, key *kms.KeyListEntry, excludeAfter time.Time) (bool, error) {
	details, err := svc.DescribeKey(&kms.DescribeKeyInput{KeyId: key.KeyId})
	if err != nil {
		return false, nil
	}
	metadata := details.KeyMetadata
	// skip keys already scheduled for removal
	if metadata.DeletionDate != nil {
		return false, nil
	}
	if *metadata.KeyManager == "CUSTOMER" {
		return false, err
	}
	var referenceTime = *metadata.CreationDate
	if referenceTime.After(excludeAfter) {
		return false, nil
	}
	return true, nil
}