package resource

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gruntwork-io/cloud-nuke/logging"
)

const (
	// DefaultMaxConcurrent is the default number of concurrent deletions
	DefaultMaxConcurrent = 10
)

// NukeResult represents the result of nuking a single resource.
type NukeResult struct {
	Identifier string
	Error      error
}

// DeleteFunc is a function that deletes a single resource by ID.
type DeleteFunc[C any] func(ctx context.Context, client C, id *string) error

// NukerFunc is the standard signature for batch deletion functions.
// Returns results for each identifier. Reporting is handled by Resource.Nuke().
type NukerFunc[C any] func(ctx context.Context, client C, scope Scope, resourceType string, identifiers []*string) []NukeResult

// logStart checks if identifiers is empty and logs the start of deletion.
// Returns true if empty (caller should return early), false otherwise.
func logStart(identifiers []*string, resourceType string, scope Scope) bool {
	if len(identifiers) == 0 {
		logging.Debugf("No %s to nuke in %s", resourceType, scope)
		return true
	}
	logging.Infof("Deleting %d %s in %s", len(identifiers), resourceType, scope)
	return false
}

// SimpleBatchDeleter creates a nuker that deletes resources concurrently.
// Uses DefaultMaxConcurrent for parallelism control.
func SimpleBatchDeleter[C any](deleteFn DeleteFunc[C]) NukerFunc[C] {
	return func(ctx context.Context, client C, scope Scope, resourceType string, identifiers []*string) []NukeResult {
		if logStart(identifiers, resourceType, scope) {
			return nil
		}

		results := make([]NukeResult, len(identifiers))
		sem := make(chan struct{}, DefaultMaxConcurrent)
		var wg sync.WaitGroup
		var mu sync.Mutex

		for i, id := range identifiers {
			wg.Add(1)
			sem <- struct{}{}

			go func(idx int, identifier *string) {
				defer wg.Done()
				defer func() { <-sem }()

				idStr := aws.ToString(identifier)
				err := deleteFn(ctx, client, identifier)

				mu.Lock()
				results[idx] = NukeResult{Identifier: idStr, Error: err}
				mu.Unlock()
			}(i, id)
		}

		wg.Wait()
		return results
	}
}

// SequentialDeleter creates a nuker that deletes resources one at a time.
// Use this for APIs with strict rate limits.
func SequentialDeleter[C any](deleteFn DeleteFunc[C]) NukerFunc[C] {
	return func(ctx context.Context, client C, scope Scope, resourceType string, identifiers []*string) []NukeResult {
		if logStart(identifiers, resourceType, scope) {
			return nil
		}

		results := make([]NukeResult, 0, len(identifiers))
		for _, id := range identifiers {
			idStr := aws.ToString(id)
			err := deleteFn(ctx, client, id)
			results = append(results, NukeResult{Identifier: idStr, Error: err})
		}

		return results
	}
}

// BulkDeleteFunc is a function that deletes multiple resources in a single API call.
type BulkDeleteFunc[C any] func(ctx context.Context, client C, ids []string) error

// BulkDeleter creates a nuker for APIs that support batch deletion in a single call.
// Use this for AWS APIs like DeleteDashboards that accept an array of identifiers.
// Note: All identifiers share the same error since it's a single API call.
func BulkDeleter[C any](deleteFn BulkDeleteFunc[C]) NukerFunc[C] {
	return BulkResultDeleter(func(ctx context.Context, client C, ids []string) []NukeResult {
		err := deleteFn(ctx, client, ids)
		results := make([]NukeResult, len(ids))
		for i, id := range ids {
			results[i] = NukeResult{Identifier: id, Error: err}
		}
		return results
	})
}

// BulkResultDeleteFunc is a function that deletes multiple resources and returns per-item results.
// Use this for AWS APIs that return partial success/failure (e.g., ReleaseHosts, DeleteMessageBatch).
type BulkResultDeleteFunc[C any] func(ctx context.Context, client C, ids []string) []NukeResult

// BulkResultDeleter creates a nuker for APIs that return per-item results in a bulk operation.
// Use this for AWS APIs where some items can succeed while others fail in the same call.
func BulkResultDeleter[C any](deleteFn BulkResultDeleteFunc[C]) NukerFunc[C] {
	return func(ctx context.Context, client C, scope Scope, resourceType string, identifiers []*string) []NukeResult {
		if logStart(identifiers, resourceType, scope) {
			return nil
		}

		ids := make([]string, len(identifiers))
		for i, id := range identifiers {
			ids[i] = aws.ToString(id)
		}

		return deleteFn(ctx, client, ids)
	}
}

// DeleteThenWait combines a delete function with a wait function into a single DeleteFunc.
// Use this with SequentialDeleter for resources that need to wait for deletion to complete.
//
// Example:
//
//	Nuker: resource.SequentialDeleter(resource.DeleteThenWait(
//	    deleteCluster,
//	    waitForClusterDeleted,
//	))
func DeleteThenWait[C any](deleteFn DeleteFunc[C], waitFn DeleteFunc[C]) DeleteFunc[C] {
	return func(ctx context.Context, client C, id *string) error {
		if err := deleteFn(ctx, client, id); err != nil {
			return err
		}
		return waitFn(ctx, client, id)
	}
}

// MultiStepDeleter creates a nuker that executes multiple steps per resource in sequence.
// Use this for resources that require cleanup before deletion (e.g., detach policies, empty bucket).
// Each resource is processed sequentially, but if any step fails for a resource, it moves to the next resource.
func MultiStepDeleter[C any](steps ...DeleteFunc[C]) NukerFunc[C] {
	return func(ctx context.Context, client C, scope Scope, resourceType string, identifiers []*string) []NukeResult {
		if logStart(identifiers, resourceType, scope) {
			return nil
		}

		results := make([]NukeResult, 0, len(identifiers))
		for _, id := range identifiers {
			idStr := aws.ToString(id)
			var stepErr error

			for i, step := range steps {
				if err := step(ctx, client, id); err != nil {
					stepErr = fmt.Errorf("step %d: %w", i+1, err)
					break
				}
			}

			results = append(results, NukeResult{Identifier: idStr, Error: stepErr})
		}

		return results
	}
}

// WaitAllFunc is a function that waits for multiple resources to be deleted.
// Used with SequentialDeleteThenWaitAll for batch waiting after all deletes complete.
type WaitAllFunc[C any] func(ctx context.Context, client C, ids []string) error

// SequentialDeleteThenWaitAll creates a nuker that:
// 1. Deletes all resources sequentially
// 2. Waits for ALL successfully deleted resources to be confirmed deleted
//
// Use this for resources where the delete API returns immediately but the resource
// takes time to be fully deleted, and the wait API can check multiple resources at once.
//
// Example:
//
//	Nuker: resource.SequentialDeleteThenWaitAll(
//	    deleteASG,
//	    waitForASGsDeleted,  // Uses autoscaling.NewGroupNotExistsWaiter
//	)
func SequentialDeleteThenWaitAll[C any](deleteFn DeleteFunc[C], waitAllFn WaitAllFunc[C]) NukerFunc[C] {
	return func(ctx context.Context, client C, scope Scope, resourceType string, identifiers []*string) []NukeResult {
		if logStart(identifiers, resourceType, scope) {
			return nil
		}

		results := make([]NukeResult, 0, len(identifiers))
		var deletedIds []string

		// Phase 1: Delete all resources sequentially
		for _, id := range identifiers {
			idStr := aws.ToString(id)
			err := deleteFn(ctx, client, id)

			if err != nil {
				results = append(results, NukeResult{Identifier: idStr, Error: err})
			} else {
				deletedIds = append(deletedIds, idStr)
				logging.Debugf("[Deleted] %s: %s (waiting for confirmation)", resourceType, idStr)
			}
		}

		// Phase 2: Wait for all successfully deleted resources
		if len(deletedIds) > 0 {
			waitErr := waitAllFn(ctx, client, deletedIds)
			for _, idStr := range deletedIds {
				results = append(results, NukeResult{Identifier: idStr, Error: waitErr})
			}
		}

		return results
	}
}

// ConcurrentDeleteThenWaitAll creates a nuker that:
// 1. Deletes all resources concurrently with controlled parallelism
// 2. Waits for ALL successfully deleted resources to be confirmed deleted
//
// Use this for resources where concurrent deletion is safe and the wait API
// can check multiple resources at once.
//
// Example:
//
//	Nuker: resource.ConcurrentDeleteThenWaitAll(
//	    deleteOpenSearchDomain,
//	    waitForOpenSearchDomainsDeleted,
//	)
func ConcurrentDeleteThenWaitAll[C any](deleteFn DeleteFunc[C], waitAllFn WaitAllFunc[C]) NukerFunc[C] {
	return func(ctx context.Context, client C, scope Scope, resourceType string, identifiers []*string) []NukeResult {
		if logStart(identifiers, resourceType, scope) {
			return nil
		}

		// Phase 1: Delete all resources concurrently
		type deleteResult struct {
			idStr string
			err   error
		}
		deleteResults := make([]deleteResult, len(identifiers))
		sem := make(chan struct{}, DefaultMaxConcurrent)
		var wg sync.WaitGroup

		for i, id := range identifiers {
			wg.Add(1)
			sem <- struct{}{}

			go func(idx int, identifier *string) {
				defer wg.Done()
				defer func() { <-sem }()

				idStr := aws.ToString(identifier)
				err := deleteFn(ctx, client, identifier)
				deleteResults[idx] = deleteResult{idStr: idStr, err: err}

				if err == nil {
					logging.Debugf("[Deleted] %s: %s (waiting for confirmation)", resourceType, idStr)
				}
			}(i, id)
		}

		wg.Wait()

		// Collect successful deletes for waiting
		var deletedIds []string
		var results []NukeResult
		for _, dr := range deleteResults {
			if dr.err != nil {
				results = append(results, NukeResult{Identifier: dr.idStr, Error: dr.err})
			} else {
				deletedIds = append(deletedIds, dr.idStr)
			}
		}

		// Phase 2: Wait for all successfully deleted resources
		if len(deletedIds) > 0 {
			waitErr := waitAllFn(ctx, client, deletedIds)
			for _, idStr := range deletedIds {
				results = append(results, NukeResult{Identifier: idStr, Error: waitErr})
			}
		}

		return results
	}
}
