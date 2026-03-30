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
var ErrInvalidGroupNotFound = errors.New("error:InvalidGroup.NotFound")
var ErrDeleteProtectionEnabled = errors.New("error:DeleteProtectionEnabled")
var ErrResourceNotFoundException = errors.New("error:ErrResourceNotFoundException")
var ErrInvalidSnapshotNotFound = errors.New("error:InvalidSnapshot.NotFound")

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
		case "InvalidGroup.NotFound":
			return ErrInvalidGroupNotFound
		case "ResourceNotFoundException":
			return ErrResourceNotFoundException
		case "InvalidSnapshot.NotFound":
			return ErrInvalidSnapshotNotFound
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

// IsThrottlingError checks if the error is an AWS API throttling error
// using structured error code matching via smithy.APIError.
func IsThrottlingError(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "RequestLimitExceeded", "ThrottlingException", "TooManyRequestsException":
			return true
		}
	}
	return false
}

// IsWarningError checks if the error is a transient/expected failure that
// should be logged as a warning rather than causing a non-zero exit code.
// These errors fall into two categories:
//
// Ordering/dependency errors — resources deleted in the wrong order. The
// dependent resource will be cleaned up on the next nuke run once the
// parent is gone:
//   - DependencyViolation: EC2 subnet/ENI/SG still referenced by another resource
//   - InvalidDBSubnetGroupStateFault: RDS subnet group in use by a DB instance
//   - InvalidDBClusterStateFault: RDS cluster can't be deleted while its instances exist
//   - InvalidClusterState: Redshift cluster has an operation in progress
//   - InvalidHomeRegionException: CloudTrail trail can only be deleted from its home region
//
// Already-deleted errors — resource was deleted between the scan and nuke
// phases (e.g., by another concurrent nuke run or TTL expiry). Safe to ignore:
//   - DBSubnetGroupNotFoundFault: RDS subnet group no longer exists
//   - DBParameterGroupNotFound: RDS parameter group no longer exists
//   - InvalidSubnetID.NotFound: EC2 subnet no longer exists
//   - InvalidNetworkInterfaceID.NotFound: EC2 ENI no longer exists
//   - InvalidDhcpOptionsID.NotFound: EC2 DHCP option set no longer exists
//   - TrailNotFoundException: CloudTrail trail already deleted by another region/job
func IsWarningError(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		// Ordering/dependency errors
		case "DependencyViolation",
			"InvalidDBSubnetGroupStateFault",
			"InvalidDBClusterStateFault",
			"InvalidClusterState",
			"InvalidHomeRegionException":
			return true
		// Already-deleted errors
		case "DBSubnetGroupNotFoundFault",
			"DBParameterGroupNotFound",
			"InvalidSubnetID.NotFound",
			"InvalidNetworkInterfaceID.NotFound",
			"InvalidDhcpOptionsID.NotFound",
			"TrailNotFoundException":
			return true
		}
	}
	return false
}
