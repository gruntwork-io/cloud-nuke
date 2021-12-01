package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/gruntwork-io/go-commons/errors"
	"time"
)

func getAllKmsKeys(session *session.Session, excludeAfter time.Time) ([]*string, error) {
	svc := kms.New(session)
	var kmsIds []*string

	input := &kms.ListKeysInput{}
	result, err := svc.ListKeys(input)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	for _, key := range result.Keys {
		key.
		kmsIds = append(kmsIds, key.KeyId)
	}


	return kmsIds, nil
}