package util

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gruntwork-io/go-commons/errors"
)

func GetCurrentAccountId(session *session.Session) (string, error) {
	stssvc := sts.New(session)

	input := &sts.GetCallerIdentityInput{}

	output, err := stssvc.GetCallerIdentity(input)
	if err != nil {
		return "", errors.WithStackTrace(err)
	}

	return aws.StringValue(output.Account), nil
}
