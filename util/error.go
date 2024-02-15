package util

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws/awserr"
)

var ErrInSufficientPermission = errors.New("error:INSUFFICIENT_PERMISSION")
var ErrDifferentOwner = errors.New("error:DIFFERENT_OWNER")

const AWsUnauthorizedError string = "UnauthorizedOperation"

// TransformAWSError
// this function is used to handle AWS errors and mapping them to a custom error message
// This could be part of a larger error-handling strategy that interacts with AWS services,
// providing a more human-readable error message for certain conditions
// ref : https://docs.aws.amazon.com/AWSEC2/latest/APIReference/errors-overview.html
func TransformAWSError(err error) error {
	if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == AWsUnauthorizedError {
		return ErrInSufficientPermission
	}
	return nil
}
