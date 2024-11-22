package util

import (
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/smithy-go"
	commonErr "github.com/gruntwork-io/go-commons/errors"
)

var ErrInSufficientPermission = errors.New("error:INSUFFICIENT_PERMISSION")
var ErrDifferentOwner = errors.New("error:DIFFERENT_OWNER")
var ErrContextExecutionTimeout = errors.New("error:EXECUTION_TIMEOUT")
var ErrInterfaceIDNotFound = errors.New("error:InterfaceIdNotFound")
var ErrInvalidPermisionNotFound = errors.New("error:InvalidPermission.NotFound")
var ErrDeleteProtectionEnabled = errors.New("error:DeleteProtectionEnabled")
var ErrResourceNotFoundException = errors.New("error:ErrResourceNotFoundException")

const AWsUnauthorizedError string = "UnauthorizedOperation"
const AWSAccessDeniedException string = "AccessDeniedException"
const AwsDryRunSuccess string = "Request would have succeeded, but DryRun flag is set."

// TransformAWSError
// this function is used to handle AWS errors and mapping them to a custom error message
// This could be part of a larger error-handling strategy that interacts with AWS services,
// providing a more human-readable error message for certain conditions
// ref : https://docs.aws.amazon.com/AWSEC2/latest/APIReference/errors-overview.html
func TransformAWSError(err error) error {
	if awsErr, ok := err.(awserr.Error); ok && (awsErr.Code() == AWsUnauthorizedError || awsErr.Code() == AWSAccessDeniedException) {
		return ErrInSufficientPermission
	}
	if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "RequestCanceled" {
		return ErrContextExecutionTimeout
	}

	// check the error is wrapped with errors.WithStackTrace(), then we can't check the actuall error is aws. So handling the situation here
	// as unwrap the error and check the underhood error type is awserr
	// NOTE : is this is not checked, then it will not print `error:EXECUTION_TIMEOUT` if the error is wrapped with WithStackTrace
	if err := commonErr.Unwrap(err); err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "RequestCanceled" {
			return ErrContextExecutionTimeout
		}
	}

	if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "InvalidNetworkInterfaceID.NotFound" {
		return ErrInterfaceIDNotFound
	}

	if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "DryRunOperation" && awsErr.Message() == AwsDryRunSuccess {
		return nil
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		if apiErr.ErrorCode() == "DryRunOperation" && apiErr.ErrorMessage() == AwsDryRunSuccess {
			return nil
		}
	}
	if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "InvalidPermission.NotFound" {
		return ErrInvalidPermisionNotFound
	}
	if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ResourceNotFoundException" {
		return ErrResourceNotFoundException
	}

	return err

}

type ResourceExecutionTimeout struct {
	Timeout time.Duration
}

func (err ResourceExecutionTimeout) Error() string {
	return fmt.Sprintf("execution timed out after: %v", err.Timeout)
}
