package util

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/smithy-go"
	commonErr "github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/require"
)

func TestTransformAWSError(t *testing.T) {
	errUnhandled := errors.New("unhandled error")
	tests := []struct {
		name    string
		argErr  error
		wantErr error
	}{
		{
			name:    "unhandled error",
			argErr:  errUnhandled,
			wantErr: errUnhandled,
		},
		{
			name: "insufficient permission",
			argErr: &smithy.GenericAPIError{
				Code:    "UnauthorizedOperation",
				Message: "UnauthorizedOperation",
			},
			wantErr: ErrInSufficientPermission,
		},
		{
			name: "AWS access denied exception",
			argErr: &smithy.GenericAPIError{
				Code:    "AccessDeniedException",
				Message: "AccessDeniedException",
			},
			wantErr: ErrInSufficientPermission,
		},
		{
			name: "request canceled",
			argErr: &smithy.GenericAPIError{
				Code:    "RequestCanceled",
				Message: "RequestCanceled",
			},
			wantErr: ErrContextExecutionTimeout,
		},
		{
			name: "wrap request canceled",
			argErr: commonErr.WithStackTrace(&smithy.GenericAPIError{
				Code:    "RequestCanceled",
				Message: "RequestCanceled",
			}),
			wantErr: ErrContextExecutionTimeout,
		},
		{
			name: "invalid network interface ID NotFound",
			argErr: &smithy.GenericAPIError{
				Code:    "InvalidNetworkInterfaceID.NotFound",
				Message: "InvalidNetworkInterfaceID.NotFound",
			},
			wantErr: ErrInterfaceIDNotFound,
		},
		{
			name: "dry run operation",
			argErr: &smithy.GenericAPIError{
				Code:    "DryRunOperation",
				Message: "Request would have succeeded, but DryRun flag is set.",
			},
			wantErr: nil,
		},
		{
			name: "invalid permission not found",
			argErr: &smithy.GenericAPIError{
				Code:    "InvalidPermission.NotFound",
				Message: "InvalidPermission.NotFound",
			},
			wantErr: ErrInvalidPermisionNotFound,
		},
		{
			name: "resource not found exception",
			argErr: &smithy.GenericAPIError{
				Code:    "ResourceNotFoundException",
				Message: "ResourceNotFoundException",
			},
			wantErr: ErrResourceNotFoundException,
		},
		{
			name: "invalid snapshot not found",
			argErr: &smithy.GenericAPIError{
				Code:    "InvalidSnapshot.NotFound",
				Message: "InvalidSnapshot.NotFound",
			},
			wantErr: ErrInvalidSnapshotNotFound,
		},
		{
			name: "smithy dry run operation",
			argErr: &smithy.GenericAPIError{
				Code:    "DryRunOperation",
				Message: "Request would have succeeded, but DryRun flag is set.",
				Fault:   smithy.FaultClient,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := TransformAWSError(tt.argErr)
			require.Equal(t, tt.wantErr, out)
		})
	}
}

func TestIsWarningError(t *testing.T) {
	warningCodes := []string{
		"DependencyViolation",
		"InvalidDBSubnetGroupStateFault",
		"InvalidDBClusterStateFault",
		"InvalidDBClusterSnapshotStateFault",
		"InvalidClusterState",
		"DBSubnetGroupNotFoundFault",
		"DBParameterGroupNotFound",
		"InvalidSubnetID.NotFound",
		"InvalidNetworkInterfaceID.NotFound",
		"InvalidHomeRegionException",
		"TrailNotFoundException",
		"CacheSubnetGroupInUse",
		"CacheSubnetGroupNotFoundFault",
		"InvalidDBSnapshotState",
		"InvalidDhcpOptionsID.NotFound",
		"InvalidCacheClusterState",
		"InvalidDBParameterGroupState",
		"AuthFailure",
		"OperationNotPermitted",
	}
	for _, code := range warningCodes {
		require.True(t, IsWarningError(&smithy.GenericAPIError{Code: code}), code)
	}

	// ENI still attached to a resource being deleted in the same run — warning
	require.True(t, IsWarningError(&smithy.GenericAPIError{
		Code:    "InvalidParameterValue",
		Message: "Network interface 'eni-038687eab4e11c405' is currently in use.",
	}))
	// Generic InvalidParameterValue without the ENI signature — NOT a warning
	require.False(t, IsWarningError(&smithy.GenericAPIError{
		Code:    "InvalidParameterValue",
		Message: "some other validation failure",
	}))

	// SCP-denied errors are warnings
	require.True(t, IsWarningError(&smithy.GenericAPIError{
		Code:    "AccessDeniedException",
		Message: "User: arn:aws:sts::123456789012:assumed-role/role/session is not authorized to perform: config:DeleteConfigurationRecorder with an explicit deny in a service control policy",
	}))

	// SCP-denied errors with different casing are still warnings
	require.True(t, IsWarningError(&smithy.GenericAPIError{
		Code:    "AccessDeniedException",
		Message: "User is not authorized with an Explicit Deny in a Service Control Policy",
	}))

	// Waiter timeout errors are warnings (deletion was initiated, will complete eventually)
	require.True(t, IsWarningError(errors.New("exceeded max wait time for NodegroupDeleted waiter")))
	require.True(t, IsWarningError(errors.New("exceeded max wait time for ClusterDeleted waiter")))
	require.True(t, IsWarningError(errors.New("exceeded max wait time for FargateProfileDeleted waiter")))

	// Wrapped waiter timeout errors are still warnings
	require.True(t, IsWarningError(fmt.Errorf("deleting nodegroup: %w", errors.New("exceeded max wait time for NodegroupDeleted waiter"))))
	require.True(t, IsWarningError(commonErr.WithStackTrace(errors.New("exceeded max wait time for NodegroupDeleted waiter"))))

	// Multierror containing only waiter timeouts is a warning
	waiterMultierr := multierror.Append(nil,
		errors.New("exceeded max wait time for NodegroupDeleted waiter"),
		errors.New("exceeded max wait time for NodegroupDeleted waiter"),
	)
	require.True(t, IsWarningError(waiterMultierr))

	// Multierror containing only warning-class errors is a warning
	mixedWarningMultierr := multierror.Append(nil,
		errors.New("exceeded max wait time for NodegroupDeleted waiter"),
		&smithy.GenericAPIError{Code: "DependencyViolation"},
	)
	require.True(t, IsWarningError(mixedWarningMultierr))

	// Multierror containing a real error alongside a waiter timeout is NOT a warning
	mixedRealMultierr := multierror.Append(nil,
		errors.New("exceeded max wait time for NodegroupDeleted waiter"),
		errors.New("some real API failure"),
	)
	require.False(t, IsWarningError(mixedRealMultierr))

	// Generic access denied errors (fixable IAM issues) are NOT warnings
	require.False(t, IsWarningError(&smithy.GenericAPIError{Code: "AccessDeniedException", Message: "User is not authorized to perform this action"}))
	require.False(t, IsWarningError(&smithy.GenericAPIError{Code: "AccessDenied"}))
	require.False(t, IsWarningError(errors.New("some error")))
	require.False(t, IsWarningError(nil))
}
