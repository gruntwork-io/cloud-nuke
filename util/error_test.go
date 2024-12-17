package util

import (
	"errors"
	"testing"

	"github.com/aws/smithy-go"
	commonErr "github.com/gruntwork-io/go-commons/errors"
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
