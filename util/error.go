package util

import (
	"errors"
	"fmt"
	"time"

	"github.com/aws/smithy-go"
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
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case AWsUnauthorizedError, AWSAccessDeniedException:
			return ErrInSufficientPermission
		case "RequestCanceled":
			return ErrContextExecutionTimeout
		case "InvalidNetworkInterfaceID.NotFound":
			return ErrInterfaceIDNotFound
		case "InvalidPermission.NotFound":
			return ErrInvalidPermisionNotFound
		case "ResourceNotFoundException":
			return ErrResourceNotFoundException
		}

		if apiErr.ErrorCode() == "DryRunOperation" && apiErr.ErrorMessage() == AwsDryRunSuccess {
			return nil
		}
	}

	return err
}

type ResourceExecutionTimeout struct {
	Timeout time.Duration
}

func (err ResourceExecutionTimeout) Error() string {
	return fmt.Sprintf("execution timed out after: %v", err.Timeout)
}
