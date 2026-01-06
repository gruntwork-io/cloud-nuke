package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"google.golang.org/api/iterator"
)

// NewGCSBuckets creates a new GCS Buckets resource using the generic resource pattern.
func NewGCSBuckets() GcpResource {
	return NewGcpResource(&resource.Resource[*storage.Client]{
		ResourceTypeName: "gcs-bucket",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapGcpInitClient(func(r *resource.Resource[*storage.Client], projectID string) {
			r.Scope.ProjectID = projectID
			client, err := storage.NewClient(context.Background())
			if err != nil {
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
		if strings.Contains(deleteErr.Error(), "bucket is not empty") {
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
func emptyBucket(ctx context.Context, bucket *storage.BucketHandle, bucketName string) error {
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
			// Continue trying to delete other objects
		}
	}
	return nil
}

// forceEmptyBucket deletes all object versions and delete markers in a bucket.
func forceEmptyBucket(ctx context.Context, bucket *storage.BucketHandle, bucketName string) error {
	it := bucket.Objects(ctx, &storage.Query{Versions: true})
	for {
		obj, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return fmt.Errorf("error listing object versions in bucket %s: %w", bucketName, err)
		}

		if err := bucket.Object(obj.Name).Generation(obj.Generation).Delete(ctx); err != nil {
			return fmt.Errorf("error deleting object version %s (gen %d) in bucket %s: %w",
				obj.Name, obj.Generation, bucketName, err)
		}
	}
	return nil
}
