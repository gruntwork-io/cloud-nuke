package util

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
)

const AWsUnauthorizedError string = "UnauthorizedOperation"

// IsAwsUnauthorizedError : checks whether the aws returned error is AWsUnauthorizedError
// For any unauthorised error we can use the same code as AWS returns the same code in this scenario
// ref : https://docs.aws.amazon.com/AWSEC2/latest/APIReference/errors-overview.html
func IsAwsUnauthorizedError(err error) bool {
	if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == AWsUnauthorizedError {
		return true
	}
	return false
}
