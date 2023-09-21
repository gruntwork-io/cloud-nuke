package util

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gruntwork-io/go-commons/errors"
)

const (
	AccountIdKey = "accountId"
)

func GetCurrentAccountId(session *session.Session) (string, error) {
	stssvc := sts.New(session)
	output, err := stssvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", errors.WithStackTrace(err)
	}

	return aws.StringValue(output.Account), nil
}
