package azure

import (
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// isThrottlingError reports whether Azure Resource Manager is rate limiting the
// caller, which ARM answers with 429 and a Retry-After header.
//
// A 503 is not one. The engine skips a batch whose every failure is a rate
// limit and records no error, so calling a service failure a rate limit loses
// the failed delete: the run reports a clean subscription over a resource group
// that is still there. The SDK has also already retried a 503 by the time the
// error reaches here, so waiting another minute answers nothing.
func isThrottlingError(err error) bool {
	var responseError *azcore.ResponseError
	if errors.As(err, &responseError) {
		return responseError.StatusCode == http.StatusTooManyRequests
	}
	return false
}
