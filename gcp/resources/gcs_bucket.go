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
	"github.com/gruntwork-io/cloud-nuke/report"
	"google.golang.org/api/iterator"
)

// getAll retrieves all GCS buckets in the project
func (b *GCSBuckets) getAll(c context.Context, configObj config.Config) ([]string, error) {
	var bucketNames []string

	it := b.Client.Buckets(c, b.ProjectID)
	for {
		bucket, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error listing buckets: %v", err)
		}

		// Check if bucket should be included based on config rules
		resourceValue := config.ResourceValue{
			Name: &bucket.Name,
			Time: &bucket.Created,
		}

		if configObj.GCSBucket.ShouldInclude(resourceValue) {
			bucketNames = append(bucketNames, bucket.Name)
			b.SetNukableStatus(bucket.Name, nil)
		}
	}

	return bucketNames, nil
}

// nukeAll deletes all GCS buckets
func (b *GCSBuckets) nukeAll(bucketNames []string) error {
	if len(bucketNames) == 0 {
		logging.Debugf("No GCS buckets to nuke")
		return nil
	}

	logging.Debugf("Deleting all GCS buckets")
	var deletedBuckets []string

	for _, bucketName := range bucketNames {
		if nukable, reason := b.IsNukable(bucketName); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", bucketName, reason)
			continue
		}

		bucket := b.Client.Bucket(bucketName)

		// Delete all objects in the bucket first
		it := bucket.Objects(b.Context, nil)
		for {
			obj, err := it.Next()
			if err != nil {
				break // End of iteration
			}
			if err := bucket.Object(obj.Name).Delete(b.Context); err != nil {
				logging.Debugf("[Failed] Error deleting object %s in bucket %s: %v", obj.Name, bucketName, err)
			}
		}

		// Delete the bucket
		if err := bucket.Delete(b.Context); err != nil {
			// Record status of this resource
			e := report.Entry{
				Identifier:   bucketName,
				ResourceType: "GCS Bucket",
				Error:        err,
			}
			report.Record(e)

			logging.Debugf("[Failed] Error deleting bucket %s: %v", bucketName, err)
		} else {
			deletedBuckets = append(deletedBuckets, bucketName)
			logging.Debugf("Deleted GCS bucket: %s", bucketName)
		}

		// Add a small delay to avoid hitting rate limits
		time.Sleep(waitDuration)
	}

	logging.Debugf("[OK] %d GCS bucket(s) deleted", len(deletedBuckets))
	return nil
}

// Nuke deletes the specified GCS buckets
func (b *GCSBuckets) Nuke(identifiers []string) error {
	if len(identifiers) == 0 {
		return nil
	}

	var lastError error
	for _, name := range identifiers {
		logging.Debugf("Deleting bucket %s", name)

		// First, delete all objects in the bucket
		bucket := b.Client.Bucket(name)
		it := bucket.Objects(b.Context, nil)
		for {
			obj, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				msg := fmt.Sprintf("Error listing objects in bucket %s: %v", name, err)
				b.SetNukableStatus(name, errors.New(msg))
				logging.Debug(msg)
				lastError = err
				continue
			}

			if err := bucket.Object(obj.Name).Delete(b.Context); err != nil {
				msg := fmt.Sprintf("Error deleting object %s in bucket %s: %v", obj.Name, name, err)
				b.SetNukableStatus(name, errors.New(msg))
				logging.Debug(msg)
				lastError = err
				continue
			}
		}

		// Then delete the bucket itself
		if err := bucket.Delete(b.Context); err != nil {
			if strings.Contains(err.Error(), "bucket is not empty") {
				// This can happen if there are object versions or delete markers
				// Try to delete with force option
				if err := b.forceDeleteBucket(name, bucket); err != nil {
					msg := fmt.Sprintf("Error force deleting bucket %s: %v", name, err)
					b.SetNukableStatus(name, errors.New(msg))
					logging.Debug(msg)
					lastError = err
					continue
				}
			} else {
				msg := fmt.Sprintf("Error deleting bucket %s: %v", name, err)
				b.SetNukableStatus(name, errors.New(msg))
				logging.Debug(msg)
				lastError = err
				continue
			}
		}

		b.SetNukableStatus(name, nil)
	}

	return lastError
}

// forceDeleteBucket attempts to delete a bucket by first removing all object versions and delete markers
func (b *GCSBuckets) forceDeleteBucket(name string, bucket *storage.BucketHandle) error {
	// List all object versions
	it := bucket.Objects(b.Context, &storage.Query{Versions: true})
	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("error listing object versions in bucket %s: %v", name, err)
		}

		// Delete the object version or delete marker
		if err := bucket.Object(obj.Name).Generation(obj.Generation).Delete(b.Context); err != nil {
			return fmt.Errorf("error deleting object version %s (gen %d) in bucket %s: %v", obj.Name, obj.Generation, name, err)
		}
	}

	// Try to delete the bucket again
	return bucket.Delete(b.Context)
}
