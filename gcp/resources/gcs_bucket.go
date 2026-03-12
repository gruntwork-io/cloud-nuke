package resources

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

// NewGCSBuckets creates a new GCS Buckets resource using the generic resource pattern.
func NewGCSBuckets() GcpResource {
	return NewGcpResource(&resource.Resource[*storage.Client]{
		ResourceTypeName: "gcs-bucket",
		BatchSize:        resource.DefaultBatchSize,
		InitClient: WrapGcpInitClient(func(r *resource.Resource[*storage.Client], cfg GcpConfig) {
			r.Scope.ProjectID = cfg.ProjectID
			client, err := storage.NewClient(context.Background())
			if err != nil {
				// Panic is recovered by GcpResourceAdapter.Init() and stored as initErr,
				// causing subsequent GetAndSetIdentifiers/Nuke calls to return the error gracefully.
				panic(fmt.Sprintf("failed to create GCS client: %v", err))
			}
			r.Client = client
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.GCSBucket
		},
		Lister: listGCSBuckets,
		Nuker:  resource.SequentialDeleter(deleteGCSBucket),
	})
}

// listGCSBuckets retrieves all GCS buckets in the project that match the config filters.
func listGCSBuckets(ctx context.Context, client *storage.Client, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var result []*string

	it := client.Buckets(ctx, scope.ProjectID)
	for {
		bucket, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error listing buckets: %w", err)
		}

		resourceValue := config.ResourceValue{
			Name: &bucket.Name,
			Time: &bucket.Created,
			Tags: bucket.Labels,
		}

		if cfg.ShouldInclude(resourceValue) {
			name := bucket.Name
			result = append(result, &name)
		}
	}

	return result, nil
}

// Rate limiting delay between bucket deletions to avoid API quota issues
const deleteDelay = 5 * time.Second

// deleteGCSBucket deletes a single GCS bucket, including all its objects.
func deleteGCSBucket(ctx context.Context, client *storage.Client, name *string) error {
	bucketName := *name
	bucket := client.Bucket(bucketName)

	// First, delete all objects in the bucket
	if err := emptyBucket(ctx, bucket, bucketName); err != nil {
		return err
	}

	// Delete the bucket
	deleteErr := bucket.Delete(ctx)
	if deleteErr != nil {
		var apiErr *googleapi.Error
		if errors.As(deleteErr, &apiErr) && apiErr.Code == 409 {
			// Bucket may have versioned objects, try force delete
			if forceErr := forceEmptyBucket(ctx, bucket, bucketName); forceErr != nil {
				return fmt.Errorf("error force emptying bucket %s: %w", bucketName, forceErr)
			}
			// Try delete again
			if retryErr := bucket.Delete(ctx); retryErr != nil {
				return fmt.Errorf("error deleting bucket %s after force empty: %w", bucketName, retryErr)
			}
		} else {
			return fmt.Errorf("error deleting bucket %s: %w", bucketName, deleteErr)
		}
	}

	logging.Debugf("Deleted GCS bucket: %s", bucketName)

	// Rate limiting delay to avoid API quota issues
	time.Sleep(deleteDelay)

	return nil
}

// emptyBucket deletes all objects in a bucket.
// Returns an error if any object deletions fail, so the caller knows the bucket may not be fully empty.
func emptyBucket(ctx context.Context, bucket *storage.BucketHandle, bucketName string) error {
	var deleteErrors int
	var lastErr error
	it := bucket.Objects(ctx, nil)
	for {
		obj, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return fmt.Errorf("error listing objects in bucket %s: %w", bucketName, err)
		}

		if err := bucket.Object(obj.Name).Delete(ctx); err != nil {
			logging.Debugf("Error deleting object %s in bucket %s: %v", obj.Name, bucketName, err)
			lastErr = err
			deleteErrors++
			// Continue trying to delete other objects
		}
	}
	if deleteErrors > 0 {
		return fmt.Errorf("failed to delete %d objects in bucket %s (last error: %w)", deleteErrors, bucketName, lastErr)
	}
	return nil
}

// forceEmptyBucket deletes all object versions and delete markers in a bucket.
// Continues on individual object deletion errors and returns the aggregate result.
func forceEmptyBucket(ctx context.Context, bucket *storage.BucketHandle, bucketName string) error {
	it := bucket.Objects(ctx, &storage.Query{Versions: true})
	var deleteErrors int
	var lastErr error
	for {
		obj, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return fmt.Errorf("error listing object versions in bucket %s: %w", bucketName, err)
		}

		if err := bucket.Object(obj.Name).Generation(obj.Generation).Delete(ctx); err != nil {
			logging.Debugf("Error deleting object version %s (gen %d) in bucket %s: %v",
				obj.Name, obj.Generation, bucketName, err)
			deleteErrors++
			lastErr = err
		}
	}
	if deleteErrors > 0 {
		return fmt.Errorf("failed to delete %d object versions in bucket %s (last error: %w)", deleteErrors, bucketName, lastErr)
	}
	return nil
}
