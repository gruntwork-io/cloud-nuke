package resource

import (
	"context"
	"fmt"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/hashicorp/go-multierror"
)

// ScanCallbacks contains hooks called during the scan loop.
// Callers wire these to reporting/telemetry without creating a dependency.
type ScanCallbacks struct {
	// OnScanProgress is called before scanning each resource type
	OnScanProgress func(resourceName, region string)
	// OnScanError is called when GetAndSetIdentifiers fails
	OnScanError func(resourceName, region string, err error)
	// OnScanComplete is called after scanning each resource type (success or failure)
	OnScanComplete func(resourceName string, count int, duration time.Duration)
	// OnResourceFound is called for each discovered resource identifier
	OnResourceFound func(resourceName, region, id string, nukable bool, reason string)
	// ShouldSkipError returns true if the error should cause the resource to be silently skipped.
	// Used for GCP service-disabled errors when the resource wasn't explicitly requested.
	// Receives the resource name and the error.
	ShouldSkipError func(resourceName string, err error) bool
}

// ScanResource scans a single resource type: calls GetAndSetResourceConfig,
// GetAndSetIdentifiers, and invokes callbacks. Returns identifiers found (nil if none/error).
func ScanResource(ctx context.Context, res NukeableResource, region string, configObj config.Config, cb ScanCallbacks) []string {
	resourceName := res.ResourceName()

	res.GetAndSetResourceConfig(configObj)

	if cb.OnScanProgress != nil {
		cb.OnScanProgress(resourceName, region)
	}

	start := time.Now()
	identifiers, err := res.GetAndSetIdentifiers(ctx, configObj)
	duration := time.Since(start)

	if err != nil {
		if cb.ShouldSkipError != nil && cb.ShouldSkipError(resourceName, err) {
			logging.Debugf("Skipping %s: %v", resourceName, err)
			return nil
		}

		logging.Errorf("Unable to retrieve %s: %v", resourceName, err)
		if cb.OnScanError != nil {
			cb.OnScanError(resourceName, region, err)
		}
	}

	if cb.OnScanComplete != nil {
		cb.OnScanComplete(resourceName, len(identifiers), duration)
	}

	if len(identifiers) > 0 {
		logging.Infof("Found %d %s resources in %s", len(identifiers), resourceName, region)

		if cb.OnResourceFound != nil {
			for _, id := range identifiers {
				nukable, reason := true, ""
				if _, nukErr := res.IsNukable(id); nukErr != nil {
					nukable, reason = false, nukErr.Error()
				}
				cb.OnResourceFound(resourceName, region, id, nukable, reason)
			}
		}
	}

	return identifiers
}

const (
	// DefaultMaxThrottleRetries is the maximum number of times to retry a batch after a throttle/quota error.
	DefaultMaxThrottleRetries = 3
	// DefaultBatchDelay is the sleep duration between batches.
	DefaultBatchDelay = 10 * time.Second
	// DefaultThrottleDelay is the sleep duration after a throttle/quota error.
	DefaultThrottleDelay = 1 * time.Minute
)

// NukeBatchCallbacks contains hooks called during the nuke batch loop.
type NukeBatchCallbacks struct {
	// OnBatchStart is called before nuking each batch
	OnBatchStart func(resourceName, region string, batchSize int)
	// OnResult is called for each individual nuke result
	OnResult func(resourceName, region string, result NukeResult)
	// OnNukeError is called when a batch nuke returns an error (after retries exhausted)
	OnNukeError func(resourceName, region string, err error)
	// IsRetryableError returns true if the error should be retried (throttle/quota errors)
	IsRetryableError func(err error) bool
}

// NukeInBatches filters non-nukable identifiers, splits into batches, and nukes them with throttle retries.
// This is the shared nuke loop used by both AWS and GCP.
func NukeInBatches(ctx context.Context, res NukeableResource, region string, cb NukeBatchCallbacks) error {
	// Filter to only nukable identifiers
	var identifiers []string
	for _, id := range res.ResourceIdentifiers() {
		nukable, reason := res.IsNukable(id)
		if !nukable {
			logging.Debugf("[Skipping] %s %s: %v", res.ResourceName(), id, reason)
			continue
		}
		identifiers = append(identifiers, id)
	}
	if len(identifiers) == 0 {
		return nil
	}

	resourceName := res.ResourceName()
	logging.Debugf("Terminating %d %s in batches", len(identifiers), resourceName)
	batches := util.Split(identifiers, res.MaxBatchSize())

	var allErrors *multierror.Error
	throttleRetries := 0

	for i := 0; i < len(batches); i++ {
		batch := batches[i]

		if cb.OnBatchStart != nil {
			cb.OnBatchStart(resourceName, region, len(batch))
		}

		results, err := res.Nuke(ctx, batch)

		// Report individual results
		if cb.OnResult != nil {
			for _, result := range results {
				cb.OnResult(resourceName, region, result)
			}
		}

		if err != nil {
			// Handle throttle/quota errors with retry limit
			if cb.IsRetryableError != nil && cb.IsRetryableError(err) && throttleRetries < DefaultMaxThrottleRetries {
				throttleRetries++
				logging.Debugf(
					"Rate limited (retry %d/%d). Waiting before making new requests",
					throttleRetries, DefaultMaxThrottleRetries,
				)
				time.Sleep(DefaultThrottleDelay)
				i--
				continue
			}

			allErrors = multierror.Append(allErrors, fmt.Errorf("[%s] %s: %w", region, resourceName, err))

			if cb.OnNukeError != nil {
				cb.OnNukeError(resourceName, region, err)
			}
		} else {
			throttleRetries = 0
		}

		if i != len(batches)-1 {
			logging.Debug("Sleeping for 10 seconds before processing next batch...")
			time.Sleep(DefaultBatchDelay)
		}
	}

	return allErrors.ErrorOrNil()
}
