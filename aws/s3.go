package aws

import (
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gruntwork-io/go-commons/errors"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
)

// getS3BucketRegion returns S3 Bucket region.
func (sb S3Buckets) getS3BucketRegion(bucketName string) (string, error) {
	input := &s3.GetBucketLocationInput{
		Bucket: aws.String(bucketName),
	}

	result, err := sb.Client.GetBucketLocation(input)
	if err != nil {
		return "", err
	}

	if result.LocationConstraint == nil {
		// GetBucketLocation returns nil for us-east-1
		// https://github.com/aws/aws-sdk-go/issues/1687
		return "us-east-1", nil
	}
	return *result.LocationConstraint, nil
}

// getS3BucketTags returns S3 Bucket tags.
func (bucket *S3Buckets) getS3BucketTags(bucketName string) ([]map[string]string, error) {
	input := &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucketName),
	}

	tags := []map[string]string{}

	// Please note that svc argument should be created from a session object which is
	// in the same region as the bucket or GetBucketTagging will fail.
	result, err := bucket.Client.GetBucketTagging(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "NoSuchTagSet":
				return tags, nil
			}
		}
		return tags, err
	}

	for _, tagSet := range result.TagSet {
		tags = append(tags, map[string]string{"Key": *tagSet.Key, "Value": *tagSet.Value})
	}

	return tags, nil
}

// hasValidTags checks if bucket tags permit it to be in the deletion list.
func hasValidTags(bucketTags []map[string]string) bool {
	// Exclude deletion of any buckets with cloud-nuke-excluded tags
	if len(bucketTags) > 0 {
		for _, tagSet := range bucketTags {
			key := strings.ToLower(tagSet["Key"])
			value := strings.ToLower(tagSet["Value"])
			if key == AwsResourceExclusionTagKey && value == "true" {
				return false
			}
		}
	}
	return true
}

// S3Bucket - represents S3 bucket
type S3Bucket struct {
	Name          string
	CreationDate  time.Time
	Tags          []map[string]string
	Error         error
	IsValid       bool
	InvalidReason string
}

// getAllS3Buckets returns a map of per region AWS S3 buckets which were created before excludeAfter
func (sb S3Buckets) getAll(configObj config.Config) ([]*string, error) {
	input := &s3.ListBucketsInput{}
	output, err := sb.Client.ListBuckets(input)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string
	totalBuckets := len(output.Buckets)
	if totalBuckets == 0 {
		return nil, nil
	}

	batchSize := sb.MaxBatchSize()
	totalBatches := int(math.Ceil(float64(totalBuckets) / float64(batchSize)))
	batchCount := 1

	// Batch the get operation
	for batchStart := 0; batchStart < totalBuckets; batchStart += batchSize {
		batchEnd := int(math.Min(float64(batchStart)+float64(batchSize), float64(totalBuckets)))
		logging.Logger.Debugf("Getting - %d-%d buckets of batch %d/%d", batchStart+1, batchEnd, batchCount, totalBatches)
		targetBuckets := output.Buckets[batchStart:batchEnd]
		bucketNames, err := sb.getBucketNames(targetBuckets, configObj)
		if err != nil {
			return nil, err
		}

		names = append(names, bucketNames...)
		batchCount++
	}

	return names, nil
}

// getBucketNamesPerRegions gets valid bucket names concurrently from list of target buckets
func (sb S3Buckets) getBucketNames(targetBuckets []*s3.Bucket, configObj config.Config) ([]*string, error) {
	var bucketNames []*string
	bucketCh := make(chan *S3Bucket, len(targetBuckets))
	var wg sync.WaitGroup

	for _, bucket := range targetBuckets {
		wg.Add(1)
		go func(bucket *s3.Bucket) {
			defer wg.Done()
			sb.getBucketInfo(bucket, bucketCh, configObj)
		}(bucket)
	}

	go func() {
		wg.Wait()
		close(bucketCh)
	}()

	// Start reading from the channel as soon as the data comes in - so that skip
	// messages are shown to the user as soon as possible
	for bucketData := range bucketCh {
		if bucketData.Error != nil {
			logging.Logger.Debugf("Skipping - Bucket %s - region - %s - error: %s", bucketData.Name, sb.Region, bucketData.Error)
			continue
		}
		if !bucketData.IsValid {
			logging.Logger.Debugf("Skipping - Bucket %s - region - %s - %s", bucketData.Name, sb.Region, bucketData.InvalidReason)
			continue
		}

		bucketNames = append(bucketNames, aws.String(bucketData.Name))
	}

	return bucketNames, nil
}

// getBucketInfo populates the local S3Bucket struct for the passed AWS bucket
func (sb S3Buckets) getBucketInfo(bucket *s3.Bucket, bucketCh chan<- *S3Bucket, configObj config.Config) {
	var bucketData S3Bucket
	bucketData.Name = aws.StringValue(bucket.Name)
	bucketData.CreationDate = aws.TimeValue(bucket.CreationDate)

	bucketRegion, err := sb.getS3BucketRegion(bucketData.Name)
	if err != nil {
		bucketData.Error = err
		bucketCh <- &bucketData
		return
	}

	// Check if the bucket is in target region
	if bucketRegion != sb.Region {
		bucketData.InvalidReason = "Not in target region"
		bucketCh <- &bucketData
		return
	}

	// Check if the bucket has valid tags
	bucketTags, err := sb.getS3BucketTags(bucketData.Name)
	if err != nil {
		bucketData.Error = err
		bucketCh <- &bucketData
		return
	}
	bucketData.Tags = bucketTags
	if !hasValidTags(bucketData.Tags) {
		bucketData.InvalidReason = "Matched tag filter"
		bucketCh <- &bucketData
		return
	}

	// Check if the bucket is older than the required time
	if !configObj.S3.ShouldInclude(config.ResourceValue{Time: &bucketData.CreationDate}) {
		bucketData.InvalidReason = "Matched CreationDate filter"
		bucketCh <- &bucketData
		return
	}

	// Check if the bucket matches config file rules
	if !configObj.S3.ShouldInclude(config.ResourceValue{Name: &bucketData.Name}) {
		bucketData.InvalidReason = "Filtered by config file rules"
		bucketCh <- &bucketData
		return
	}

	bucketData.IsValid = true
	bucketCh <- &bucketData
}

// emptyBucket will empty the given S3 bucket by deleting all the objects that are in the bucket. For versioned buckets,
// this includes all the versions and deletion markers in the bucket.
// NOTE: In the progress logs, we deliberately do not report how many pages or objects are left. This is because aws
// does not provide any API for getting the object count, and the only way to do that is to iterate through all the
// objects. For memory and time efficiency, we opted to delete the objects as we retrieve each page, which means we
// don't know how many are left until we complete all the operations.
func (sb S3Buckets) emptyBucket(bucketName *string, isVersioned bool) error {
	// Since the error may happen in the inner function handler for the pager, we need a function scoped variable that
	// the inner function can set when there is an error.
	var errOut error
	pageId := 1

	// Handle versioned buckets.
	if isVersioned {
		err := sb.Client.ListObjectVersionsPages(
			&s3.ListObjectVersionsInput{
				Bucket:  bucketName,
				MaxKeys: aws.Int64(int64(sb.MaxBatchSize())),
			},
			func(page *s3.ListObjectVersionsOutput, lastPage bool) (shouldContinue bool) {
				logging.Logger.Debugf("Deleting page %d of object versions (%d objects) from bucket %s", pageId, len(page.Versions), aws.StringValue(bucketName))
				if err := sb.deleteObjectVersions(bucketName, page.Versions); err != nil {
					logging.Logger.Errorf("Error deleting objects versions for page %d from bucket %s: %s", pageId, aws.StringValue(bucketName), err)
					errOut = err
					return false
				}
				logging.Logger.Debugf("[OK] - deleted page %d of object versions (%d objects) from bucket %s", pageId, len(page.Versions), aws.StringValue(bucketName))

				logging.Logger.Debugf("Deleting page %d of deletion markers (%d deletion markers) from bucket %s", pageId, len(page.DeleteMarkers), aws.StringValue(bucketName))
				if err := sb.deleteDeletionMarkers(bucketName, page.DeleteMarkers); err != nil {
					logging.Logger.Debugf("Error deleting deletion markers for page %d from bucket %s: %s", pageId, aws.StringValue(bucketName), err)
					errOut = err
					return false
				}
				logging.Logger.Debugf("[OK] - deleted page %d of deletion markers (%d deletion markers) from bucket %s", pageId, len(page.DeleteMarkers), aws.StringValue(bucketName))

				pageId++
				return true
			},
		)
		if err != nil {
			return err
		}
		if errOut != nil {
			return errOut
		}
		return nil
	}

	// Handle non versioned buckets.
	err := sb.Client.ListObjectsV2Pages(
		&s3.ListObjectsV2Input{
			Bucket:  bucketName,
			MaxKeys: aws.Int64(int64(sb.MaxBatchSize())),
		},
		func(page *s3.ListObjectsV2Output, lastPage bool) (shouldContinue bool) {
			logging.Logger.Debugf("Deleting object page %d (%d objects) from bucket %s", pageId, len(page.Contents), aws.StringValue(bucketName))
			if err := sb.deleteObjects(bucketName, page.Contents); err != nil {
				logging.Logger.Errorf("Error deleting objects for page %d from bucket %s: %s", pageId, aws.StringValue(bucketName), err)
				errOut = err
				return false
			}
			logging.Logger.Debugf("[OK] - deleted object page %d (%d objects) from bucket %s", pageId, len(page.Contents), aws.StringValue(bucketName))

			pageId++
			return true
		},
	)
	if err != nil {
		return err
	}
	if errOut != nil {
		return errOut
	}
	return nil
}

// deleteObjects will delete the provided objects (unversioned) from the specified bucket.
func (sb S3Buckets) deleteObjects(bucketName *string, objects []*s3.Object) error {
	if len(objects) == 0 {
		logging.Logger.Debugf("No objects returned in page")
		return nil
	}

	objectIdentifiers := []*s3.ObjectIdentifier{}
	for _, obj := range objects {
		objectIdentifiers = append(objectIdentifiers, &s3.ObjectIdentifier{
			Key: obj.Key,
		})
	}
	_, err := sb.Client.DeleteObjects(
		&s3.DeleteObjectsInput{
			Bucket: bucketName,
			Delete: &s3.Delete{
				Objects: objectIdentifiers,
				Quiet:   aws.Bool(false),
			},
		},
	)
	return err
}

// deleteObjectVersions will delete the provided object versions from the specified bucket.
func (sb S3Buckets) deleteObjectVersions(bucketName *string, objectVersions []*s3.ObjectVersion) error {
	if len(objectVersions) == 0 {
		logging.Logger.Debugf("No object versions returned in page")
		return nil
	}

	objectIdentifiers := []*s3.ObjectIdentifier{}
	for _, obj := range objectVersions {
		objectIdentifiers = append(objectIdentifiers, &s3.ObjectIdentifier{
			Key:       obj.Key,
			VersionId: obj.VersionId,
		})
	}
	_, err := sb.Client.DeleteObjects(
		&s3.DeleteObjectsInput{
			Bucket: bucketName,
			Delete: &s3.Delete{
				Objects: objectIdentifiers,
				Quiet:   aws.Bool(false),
			},
		},
	)
	return err
}

// deleteDeletionMarkers will delete the provided deletion markers from the specified bucket.
func (sb S3Buckets) deleteDeletionMarkers(bucketName *string, objectDelMarkers []*s3.DeleteMarkerEntry) error {
	if len(objectDelMarkers) == 0 {
		logging.Logger.Debugf("No deletion markers returned in page")
		return nil
	}

	objectIdentifiers := []*s3.ObjectIdentifier{}
	for _, obj := range objectDelMarkers {
		objectIdentifiers = append(objectIdentifiers, &s3.ObjectIdentifier{
			Key:       obj.Key,
			VersionId: obj.VersionId,
		})
	}
	_, err := sb.Client.DeleteObjects(
		&s3.DeleteObjectsInput{
			Bucket: bucketName,
			Delete: &s3.Delete{
				Objects: objectIdentifiers,
				Quiet:   aws.Bool(false),
			},
		},
	)
	return err
}

// nukeAllS3BucketObjects batch deletes all objects in an S3 bucket
func (sb S3Buckets) nukeAllS3BucketObjects(bucketName *string) error {
	versioningResult, err := sb.Client.GetBucketVersioning(&s3.GetBucketVersioningInput{
		Bucket: bucketName,
	})
	if err != nil {
		return err
	}

	isVersioned := aws.StringValue(versioningResult.Status) == "Enabled"

	if sb.MaxBatchSize() < 1 || sb.MaxBatchSize() > 1000 {
		return fmt.Errorf("Invalid batchsize - %d - should be between %d and %d", sb.MaxBatchSize(), 1, 1000)
	}

	logging.Logger.Debugf("Emptying bucket %s", aws.StringValue(bucketName))
	if err := sb.emptyBucket(bucketName, isVersioned); err != nil {
		return err
	}
	logging.Logger.Debugf("[OK] - successfully emptied bucket %s", aws.StringValue(bucketName))
	return nil
}

// nukeEmptyS3Bucket deletes an empty S3 bucket
func (sb S3Buckets) nukeEmptyS3Bucket(bucketName *string, verifyBucketDeletion bool) error {
	_, err := sb.Client.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: bucketName,
	})
	if err != nil {
		return err
	}

	if !verifyBucketDeletion {
		return err
	}

	// The wait routine will try for up to 100 seconds, but that is not long enough for all circumstances of S3. As
	// such, we retry this routine up to 3 times for a total of 300 seconds.
	const maxRetries = 3
	for i := 0; i < maxRetries; i++ {
		logging.Logger.Debugf("Waiting until bucket (%s) deletion is propagated (attempt %d / %d)", aws.StringValue(bucketName), i+1, maxRetries)
		err = sb.Client.WaitUntilBucketNotExists(&s3.HeadBucketInput{
			Bucket: bucketName,
		})
		// Exit early if no error
		if err == nil {
			logging.Logger.Debug("Successfully detected bucket deletion.")
			return nil
		}
		logging.Logger.Debugf("Error waiting for bucket (%s) deletion propagation (attempt %d / %d)", aws.StringValue(bucketName), i+1, maxRetries)
		logging.Logger.Debugf("Underlying error was: %s", err)
	}
	return err
}

func (sb S3Buckets) nukeS3BucketPolicy(bucketName *string) error {
	_, err := sb.Client.DeleteBucketPolicy(&s3.DeleteBucketPolicyInput{
		Bucket: aws.String(*bucketName),
	})
	return err
}

// nukeAllS3Buckets deletes all S3 buckets passed as input
func (sb S3Buckets) nukeAll(bucketNames []*string) (delCount int, err error) {
	verifyBucketDeletion := true

	if len(bucketNames) == 0 {
		logging.Logger.Debugf("No S3 Buckets to nuke in region %s", sb.Region)
		return 0, nil
	}

	totalCount := len(bucketNames)

	logging.Logger.Debugf("Deleting - %d S3 Buckets in region %s", totalCount, sb.Region)

	multiErr := new(multierror.Error)
	for bucketIndex := 0; bucketIndex < totalCount; bucketIndex++ {

		bucketName := bucketNames[bucketIndex]
		logging.Logger.Debugf("Deleting - %d/%d - Bucket: %s", bucketIndex+1, totalCount, *bucketName)

		err = sb.nukeAllS3BucketObjects(bucketName)
		if err != nil {
			logging.Logger.Debugf("[Failed] - %d/%d - Bucket: %s - object deletion error - %s", bucketIndex+1, totalCount, *bucketName, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking S3 Bucket Objects",
			}, map[string]interface{}{
				"region": sb.Region,
			})
			multierror.Append(multiErr, err)
			continue
		}

		err = sb.nukeS3BucketPolicy(bucketName)
		if err != nil {
			logging.Logger.Debugf("[Failed] - %d/%d - Bucket: %s - bucket policy cleanup error - %s", bucketIndex+1, totalCount, *bucketName, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking S3 Bucket Polikcy",
			}, map[string]interface{}{
				"region": sb.Region,
			})
			multierror.Append(multiErr, err)
			continue
		}

		err = sb.nukeEmptyS3Bucket(bucketName, verifyBucketDeletion)
		if err != nil {
			logging.Logger.Debugf("[Failed] - %d/%d - Bucket: %s - bucket deletion error - %s", bucketIndex+1, totalCount, *bucketName, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking S3 Bucket",
			}, map[string]interface{}{
				"region": sb.Region,
			})
			multierror.Append(multiErr, err)
			continue
		}

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(bucketName),
			ResourceType: "S3 Bucket",
			Error:        multiErr.ErrorOrNil(),
		}
		report.Record(e)

		logging.Logger.Debugf("[OK] - %d/%d - Bucket: %s - deleted", bucketIndex+1, totalCount, *bucketName)
		delCount++
	}

	return delCount, multiErr.ErrorOrNil()
}
