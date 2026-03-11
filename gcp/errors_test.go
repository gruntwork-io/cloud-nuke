package gcp

import (
	"fmt"
	"testing"

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
