package resource

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/hashicorp/go-multierror"
)

const (
	// DefaultMaxConcurrent is the default number of concurrent deletions
	DefaultMaxConcurrent = 10

	// MaxBatchSizeLimit is the maximum number of resources that can be deleted in a single batch
	// to avoid hitting API rate limits
	MaxBatchSizeLimit = 100
)

// DeleteFunc is a function that deletes a single resource by ID.
type DeleteFunc[C any] func(ctx context.Context, client C, id *string) error

// NukerFunc is the standard signature for batch deletion functions.
// This is what Resource.Nuker expects.
type NukerFunc[C any] func(ctx context.Context, client C, scope Scope, resourceType string, identifiers []*string) error

// logEmptyAndSkip logs that there are no resources to delete and returns true.
// Use this at the start of nuker functions.
func logEmptyAndSkip(identifiers []*string, resourceType string, scope Scope) bool {
	if len(identifiers) == 0 {
		logging.Debugf("No %s to nuke in %s", resourceType, scope)
		return true
	}
	return false
}

// logDeletionStart logs the start of a deletion operation.
func logDeletionStart(count int, resourceType string, scope Scope) {
	logging.Infof("Deleting %d %s in %s", count, resourceType, scope)
}

// SimpleBatchDeleter creates a nuker that deletes resources concurrently.
// Uses DefaultMaxConcurrent for parallelism control.
func SimpleBatchDeleter[C any](deleteFn DeleteFunc[C]) NukerFunc[C] {
	return func(ctx context.Context, client C, scope Scope, resourceType string, identifiers []*string) error {
		if logEmptyAndSkip(identifiers, resourceType, scope) {
			return nil
		}

		// Guard against too many requests that could cause rate limiting
		if len(identifiers) > MaxBatchSizeLimit {
			logging.Errorf("Nuking too many %s at once (%d): halting to avoid hitting rate limiting",
				resourceType, len(identifiers))
			return fmt.Errorf("too many %s requested at once (%d > %d limit)", resourceType, len(identifiers), MaxBatchSizeLimit)
		}

		logDeletionStart(len(identifiers), resourceType, scope)

		// Semaphore for concurrency control
		sem := make(chan struct{}, DefaultMaxConcurrent)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var allErrs *multierror.Error

		for _, id := range identifiers {
			wg.Add(1)
			// Acquire semaphore slot
			sem <- struct{}{}

			go func(identifier *string) {
				defer wg.Done()
				// Release semaphore slot when done
				defer func() { <-sem }()

				idStr := aws.ToString(identifier)
				err := deleteFn(ctx, client, identifier)

				mu.Lock()
				report.Record(report.Entry{
					Identifier:   idStr,
					ResourceType: resourceType,
					Error:        err,
				})
				if err != nil {
					logging.Errorf("[Failed] %s %s: %s", resourceType, idStr, err)
					allErrs = multierror.Append(allErrs, fmt.Errorf("%s %s: %w", resourceType, idStr, err))
				} else {
					logging.Debugf("[OK] Deleted %s: %s", resourceType, idStr)
				}
				mu.Unlock()
			}(id)
		}

		wg.Wait()
		return allErrs.ErrorOrNil()
	}
}

// SequentialDeleter creates a nuker that deletes resources one at a time.
// Use this for APIs with strict rate limits.
func SequentialDeleter[C any](deleteFn DeleteFunc[C]) NukerFunc[C] {
	return func(ctx context.Context, client C, scope Scope, resourceType string, identifiers []*string) error {
		if logEmptyAndSkip(identifiers, resourceType, scope) {
			return nil
		}

		logDeletionStart(len(identifiers), resourceType, scope)

		var allErrs *multierror.Error

		for _, id := range identifiers {
			idStr := aws.ToString(id)
			err := deleteFn(ctx, client, id)

			report.Record(report.Entry{
				Identifier:   idStr,
				ResourceType: resourceType,
				Error:        err,
			})
			if err != nil {
				logging.Errorf("[Failed] %s %s: %s", resourceType, idStr, err)
				allErrs = multierror.Append(allErrs, fmt.Errorf("%s %s: %w", resourceType, idStr, err))
			} else {
				logging.Debugf("[OK] Deleted %s: %s", resourceType, idStr)
			}
		}

		return allErrs.ErrorOrNil()
	}
}

// BulkDeleteFunc is a function that deletes multiple resources in a single API call.
type BulkDeleteFunc[C any] func(ctx context.Context, client C, ids []string) error

// BulkDeleter creates a nuker for APIs that support batch deletion in a single call.
// Use this for AWS APIs like DeleteDashboards that accept an array of identifiers.
func BulkDeleter[C any](deleteFn BulkDeleteFunc[C]) NukerFunc[C] {
	return func(ctx context.Context, client C, scope Scope, resourceType string, identifiers []*string) error {
		if logEmptyAndSkip(identifiers, resourceType, scope) {
			return nil
		}

		if len(identifiers) > MaxBatchSizeLimit {
			logging.Errorf("Nuking too many %s at once (%d): halting to avoid hitting rate limiting",
				resourceType, len(identifiers))
			return fmt.Errorf("too many %s requested at once (%d > %d limit)", resourceType, len(identifiers), MaxBatchSizeLimit)
		}

		logDeletionStart(len(identifiers), resourceType, scope)

		ids := make([]string, len(identifiers))
		for i, id := range identifiers {
			ids[i] = aws.ToString(id)
		}

		err := deleteFn(ctx, client, ids)

		report.RecordBatch(report.BatchEntry{
			Identifiers:  ids,
			ResourceType: resourceType,
			Error:        err,
		})

		if err != nil {
			logging.Errorf("[Failed] %s: %s", resourceType, err)
			return err
		}

		for _, id := range ids {
			logging.Debugf("[OK] Deleted %s: %s", resourceType, id)
		}
		return nil
	}
}

// MultiStepDeleter creates a nuker that executes multiple steps per resource in sequence.
// Use this for resources that require cleanup before deletion (e.g., detach policies, empty bucket).
// Each resource is processed sequentially, but if any step fails for a resource, it moves to the next resource.
func MultiStepDeleter[C any](steps ...DeleteFunc[C]) NukerFunc[C] {
	return func(ctx context.Context, client C, scope Scope, resourceType string, identifiers []*string) error {
		if logEmptyAndSkip(identifiers, resourceType, scope) {
			return nil
		}

		logDeletionStart(len(identifiers), resourceType, scope)

		var allErrs *multierror.Error

		for _, id := range identifiers {
			idStr := aws.ToString(id)
			var stepErr error

			for i, step := range steps {
				if err := step(ctx, client, id); err != nil {
					logging.Errorf("[Failed] %s %s step %d: %s", resourceType, idStr, i+1, err)
					stepErr = fmt.Errorf("%s %s step %d: %w", resourceType, idStr, i+1, err)
					break
				}
			}

			report.Record(report.Entry{
				Identifier:   idStr,
				ResourceType: resourceType,
				Error:        stepErr,
			})
			if stepErr != nil {
				allErrs = multierror.Append(allErrs, stepErr)
			} else {
				logging.Debugf("[OK] Deleted %s: %s", resourceType, idStr)
			}
		}

		return allErrs.ErrorOrNil()
	}
}

