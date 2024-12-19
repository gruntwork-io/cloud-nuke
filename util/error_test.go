package util

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws/awserr"
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
			name:    "insufficient permission",
			argErr:  awserr.New("UnauthorizedOperation", "UnauthorizedOperation", nil),
			wantErr: ErrInSufficientPermission,
		},
		{
			name:    "AWS access denied exception",
			argErr:  awserr.New("AccessDeniedException", "AccessDeniedException", nil),
			wantErr: ErrInSufficientPermission,
		},
		{
			name:    "request canceled",
			argErr:  awserr.New("RequestCanceled", "RequestCanceled", nil),
			wantErr: ErrContextExecutionTimeout,
		},
		{
			name:    "wrap request canceled",
			argErr:  commonErr.WithStackTrace(awserr.New("RequestCanceled", "RequestCanceled", nil)),
			wantErr: ErrContextExecutionTimeout,
		},
		{
			name:    "invalid network interface ID NotFound",
			argErr:  awserr.New("InvalidNetworkInterfaceID.NotFound", "InvalidNetworkInterfaceID.NotFound", nil),
			wantErr: ErrInterfaceIDNotFound,
		},
		{
			name:    "dry run operation",
			argErr:  awserr.New("DryRunOperation", "Request would have succeeded, but DryRun flag is set.", nil),
			wantErr: nil,
		},
		{
			name:    "invalid permission not found",
			argErr:  awserr.New("InvalidPermission.NotFound", "InvalidPermission.NotFound", nil),
			wantErr: ErrInvalidPermisionNotFound,
		},
		{
			name:    "resource not found exception",
			argErr:  awserr.New("ResourceNotFoundException", "ResourceNotFoundException", nil),
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
