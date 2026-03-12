package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

// isGCPNotFound returns true if the error is a GCP API 404 Not Found error.
func isGCPNotFound(err error) bool {
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return apiErr.Code == 404
	}
	return false
}

// waitForZoneOperation waits for a zone operation to complete using the GCP
// long-poll Wait API. If the provided context has no deadline, a default
// timeout of DefaultWaitTimeout is applied to prevent indefinite blocking.
func waitForZoneOperation(ctx context.Context, client *compute.Service, project, zone, opName string) error {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultWaitTimeout)
		defer cancel()
	}

	for {
		op, err := client.ZoneOperations.Wait(project, zone, opName).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("error waiting for operation %s: %w", opName, err)
		}
		if op.Status == "DONE" {
			if op.Error != nil && len(op.Error.Errors) > 0 {
				msgs := make([]string, len(op.Error.Errors))
				for i, e := range op.Error.Errors {
					msgs[i] = e.Message
				}
				return fmt.Errorf("operation %s failed: %s", opName, strings.Join(msgs, "; "))
			}
			return nil
		}
		// ZoneOperations.Wait returns when the operation completes or the
		// server-side timeout (default ~2 min) elapses. If it returned
		// without DONE, loop to wait again.
	}
}
