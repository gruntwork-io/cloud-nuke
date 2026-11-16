package util

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// AccountIdKey is the context key used to store the AWS account ID.
const AccountIdKey ContextKey = "accountId"

func GetCurrentAccountId(config aws.Config) (string, error) {
	stssvc := sts.NewFromConfig(config)
	output, err := stssvc.GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}

	return aws.ToString(output.Account), nil
}
