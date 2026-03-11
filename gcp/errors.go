package gcp

import (
	"errors"

	"google.golang.org/api/googleapi"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// isServiceDisabledError checks whether the error (or any wrapped cause) is a
// gRPC SERVICE_DISABLED error from googleapis.com. It walks the error chain
// because by the time errors reach this layer they have been wrapped by
// intermediate callers (resource listers, resource.go, etc.) and
// status.FromError only inspects the outermost error.
func isServiceDisabledError(err error) bool {
	for current := err; current != nil; current = errors.Unwrap(current) {
		s, ok := status.FromError(current)
		if !ok || s.Code() == codes.OK {
			continue
		}
		for _, detail := range s.Details() {
			if info, ok := detail.(*errdetails.ErrorInfo); ok {
				if info.Reason == "SERVICE_DISABLED" && info.Domain == "googleapis.com" {
					return true
				}
			}
		}
	}
	return false
}

// isQuotaExhaustedError checks whether the error represents a GCP quota
// exceeded / rate-limit error using structured gRPC and HTTP error types
// instead of fragile string matching.
func isQuotaExhaustedError(err error) bool {
	// Check gRPC status code (ResourceExhausted = quota/rate-limit)
	for current := err; current != nil; current = errors.Unwrap(current) {
		s, ok := status.FromError(current)
		if !ok || s.Code() == codes.OK {
			continue
		}
		if s.Code() == codes.ResourceExhausted {
			return true
		}
	}

	// Check Google HTTP API errors:
	// - 429 (Too Many Requests) is the standard rate-limit response
	// - 403 with rateLimitExceeded/userRateLimitExceeded is also used for quota errors
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		if apiErr.Code == 429 {
			return true
		}
		if apiErr.Code == 403 {
			for _, e := range apiErr.Errors {
				if e.Reason == "rateLimitExceeded" || e.Reason == "userRateLimitExceeded" {
					return true
				}
			}
		}
	}

	return false
}
