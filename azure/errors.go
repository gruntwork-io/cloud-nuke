package azure

import (
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// isThrottlingError reports whether Azure Resource Manager is rate limiting the
// caller. ARM answers 429 with a Retry-After header, and occasionally 503 while
// a region is under load, which the caller should also wait out rather than
// record as a failed delete.
func isThrottlingError(err error) bool {
	var responseError *azcore.ResponseError
	if errors.As(err, &responseError) {
		switch responseError.StatusCode {
		case http.StatusTooManyRequests, http.StatusServiceUnavailable:
			return true
		}
	}
	return false
}
