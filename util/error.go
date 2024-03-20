package util

import (
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
)

var ErrInSufficientPermission = errors.New("error:INSUFFICIENT_PERMISSION")
var ErrDifferentOwner = errors.New("error:DIFFERENT_OWNER")
var ErrContextExecutionTimeout = errors.New("error:EXECUTION_TIMEOUT")

const AWsUnauthorizedError string = "UnauthorizedOperation"
const AwsDryrunSuccess string = "Request would have succeeded, but DryRun flag is set."

// TransformAWSError
// this function is used to handle AWS errors and mapping them to a custom error message
// This could be part of a larger error-handling strategy that interacts with AWS services,
// providing a more human-readable error message for certain conditions
// ref : https://docs.aws.amazon.com/AWSEC2/latest/APIReference/errors-overview.html
func TransformAWSError(err error) error {
	if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == AWsUnauthorizedError {
		return ErrInSufficientPermission
	}
	if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "RequestCanceled" {
		return ErrContextExecutionTimeout
	}

	if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "DryRunOperation" && awsErr.Message() == AwsDryrunSuccess {
		return nil
	}
	return err

}

type ResourceExecutionTimeout struct {
	Timeout time.Duration
}

func (err ResourceExecutionTimeout) Error() string {
	return fmt.Sprintf("execution timed out after: %v", err.Timeout)
}

// Check if the error is due to context deadline exceeded
func CheckDeadlineExceeded(err error) bool {
	if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "RequestCanceled" {
		return true
	}
	return false
}
