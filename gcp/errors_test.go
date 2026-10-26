package gcp

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/googleapi"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func makeServiceDisabledErr() error {
	st := status.New(codes.PermissionDenied, "Cloud Functions API has not been used in project 123 before or it is disabled.")
	st, _ = st.WithDetails(&errdetails.ErrorInfo{
		Reason: "SERVICE_DISABLED",
		Domain: "googleapis.com",
		Metadata: map[string]string{
			"consumer": "projects/123",
			"service":  "cloudfunctions.googleapis.com",
		},
	})
	return st.Err()
}

func TestIsServiceDisabledError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"direct SERVICE_DISABLED", makeServiceDisabledErr(), true},
		{"wrapped SERVICE_DISABLED", fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", makeServiceDisabledErr())), true},
		{"other gRPC error", status.New(codes.Internal, "fail").Err(), false},
		{"plain error", fmt.Errorf("random"), false},
		{"nil", nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isServiceDisabledError(tc.err))
		})
	}
}

func TestIsQuotaExhaustedError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"gRPC ResourceExhausted", status.New(codes.ResourceExhausted, "quota").Err(), true},
		{"wrapped gRPC ResourceExhausted", fmt.Errorf("nuke: %w", status.New(codes.ResourceExhausted, "quota").Err()), true},
		{"HTTP 429", &googleapi.Error{Code: 429}, true},
		{"wrapped HTTP 429", fmt.Errorf("fail: %w", &googleapi.Error{Code: 429}), true},
		{"HTTP 403 rateLimitExceeded", &googleapi.Error{Code: 403, Errors: []googleapi.ErrorItem{{Reason: "rateLimitExceeded"}}}, true},
		{"HTTP 403 userRateLimitExceeded", &googleapi.Error{Code: 403, Errors: []googleapi.ErrorItem{{Reason: "userRateLimitExceeded"}}}, true},
		{"HTTP 403 other reason", &googleapi.Error{Code: 403, Errors: []googleapi.ErrorItem{{Reason: "forbidden"}}}, false},
		{"other gRPC error", status.New(codes.Internal, "fail").Err(), false},
		{"HTTP 500", &googleapi.Error{Code: 500}, false},
		{"plain error", fmt.Errorf("random"), false},
		{"nil", nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isQuotaExhaustedError(tc.err))
		})
	}
}

// TestIsServiceDisabledErrorReadsTheRestShape pins that a disabled API is
// recognised when the REST clients report it. A GCS bucket listing answers with
// a googleapi.Error rather than a gRPC status, so reading only the gRPC shape
// reported a switched-off Cloud Storage API as a failure instead of skipping it.
func TestIsServiceDisabledErrorReadsTheRestShape(t *testing.T) {
	t.Parallel()

	byReason := &googleapi.Error{
		Code: 403,
		Errors: []googleapi.ErrorItem{
			{Reason: "accessNotConfigured", Message: "Cloud Storage API has not been used in project 1 before or it is disabled"},
		},
	}
	assert.True(t, isServiceDisabledError(byReason))

	// Wrapped the way the resource layer wraps what a lister returned.
	assert.True(t, isServiceDisabledError(fmt.Errorf("gcs-bucket: failed to list resources: %w", byReason)))

	// A plain permission error is not a disabled API, and must still be reported.
	forbidden := &googleapi.Error{
		Code:   403,
		Errors: []googleapi.ErrorItem{{Reason: "forbidden", Message: "does not have storage.buckets.list access"}},
	}
	assert.False(t, isServiceDisabledError(forbidden))

	// Neither is a quota error, which has its own handling.
	assert.False(t, isServiceDisabledError(&googleapi.Error{Code: 429}))
}
