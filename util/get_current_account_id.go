package util

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/gruntwork-io/go-commons/errors"
)

// contextKey is an unexported type for context keys to avoid collisions.
type contextKey string

// AccountIdKey is the context key used to store the AWS account ID.
var AccountIdKey = contextKey("accountId")

func GetCurrentAccountId(config aws.Config) (string, error) {
	stssvc := sts.NewFromConfig(config)
	output, err := stssvc.GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", errors.WithStackTrace(err)
	}

	return aws.ToString(output.Account), nil
}
