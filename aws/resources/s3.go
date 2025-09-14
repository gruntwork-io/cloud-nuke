package resources

import (
	"context"
	goerr "errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/hashicorp/go-multierror"
)

const AwsResourceExclusionTagKey = "cloud-nuke-excluded"

// getS3BucketRegion returns S3 Bucket region.
func (sb S3Buckets) getS3BucketRegion(bucketName string) (string, error) {
	input := &s3.GetBucketLocationInput{
		Bucket: aws.String(bucketName),
	}

	result, err := sb.Client.GetBucketLocation(sb.Context, input)
	if err != nil {
		return "", err
	}

	if result.LocationConstraint == "" {
		// GetBucketLocation returns nil for us-east-1
		// https://github.com/aws/aws-sdk-go/issues/1687
		return "us-east-1", nil
	}
	return string(result.LocationConstraint), nil
}

// getS3BucketTags returns S3 Bucket tags.
func (bucket *S3Buckets) getS3BucketTags(bucketName string) (map[string]string, error) {
	input := &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucketName),
	}

	// Please note that svc argument should be created from a session object which is
	// in the same region as the bucket or GetBucketTagging will fail.
	result, err := bucket.Client.GetBucketTagging(bucket.Context, input)
	if err != nil {
		var apiErr *smithy.OperationError
		if goerr.As(err, &apiErr) {
			if strings.Contains(apiErr.Error(), "NoSuchTagSet: The TagSet does not exist") {
				return nil, nil
			}
			return nil, err
		}
	}

	return util.ConvertS3TypesTagsToMap(result.TagSet), nil
}

// S3Bucket - represents S3 bucket
type S3Bucket struct {
	Name          string
	CreationDate  time.Time
	Tags          map[string]string
	Error         error
	IsValid       bool
	InvalidReason string
}

// getAllS3Buckets returns a map of per region AWS S3 buckets which were created before excludeAfter
func (sb S3Buckets) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	output, err := sb.Client.ListBuckets(sb.Context, &s3.ListBucketsInput{})
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
		logging.Debugf("Getting - %d-%d buckets of batch %d/%d", batchStart+1, batchEnd, batchCount, totalBatches)
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
func (sb S3Buckets) getBucketNames(targetBuckets []types.Bucket, configObj config.Config) ([]*string, error) {
	var bucketNames []*string
	bucketCh := make(chan *S3Bucket, len(targetBuckets))
	var wg sync.WaitGroup

	for _, bucket := range targetBuckets {
		wg.Add(1)
		go func(bucket types.Bucket) {
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
			logging.Debugf("Skipping - Bucket %s - region - %s - error: %s", bucketData.Name, sb.Region, bucketData.Error)
			continue
		}
		if !bucketData.IsValid {
			logging.Debugf("Skipping - Bucket %s - region - %s - %s", bucketData.Name, sb.Region, bucketData.InvalidReason)
			continue
		}

		bucketNames = append(bucketNames, aws.String(bucketData.Name))
	}

	return bucketNames, nil
}

// getBucketInfo populates the local S3Bucket struct for the passed AWS bucket
func (sb S3Buckets) getBucketInfo(bucket types.Bucket, bucketCh chan<- *S3Bucket, configObj config.Config) {
	var bucketData S3Bucket
	bucketData.Name = aws.ToString(bucket.Name)
	bucketData.CreationDate = aws.ToTime(bucket.CreationDate)

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
	if !configObj.S3.ShouldInclude(config.ResourceValue{
		Time: &bucketData.CreationDate,
		Name: &bucketData.Name,
		Tags: bucketTags,
	}) {
		bucketData.InvalidReason = "filtered"
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
func (sb S3Buckets) emptyBucket(bucketName *string) error {
	// Since the error may happen in the inner function handler for the pager, we need a function scoped variable that
	// the inner function can set when there is an error.
	pageId := 1

	// As bucket versioning is managed separately and you can turn off versioning after the bucket is created,
	// we need to check if there are any versions in the bucket regardless of the versioning status.
	// We need to paginate through ALL versioned objects and deletion markers.

	var keyMarker *string
	var versionIdMarker *string

	// Paginate through all versioned objects and deletion markers
	for {
		input := &s3.ListObjectVersionsInput{
			Bucket:  bucketName,
			MaxKeys: aws.Int32(int32(sb.MaxBatchSize())),
		}

		if keyMarker != nil {
			input.KeyMarker = keyMarker
		}
		if versionIdMarker != nil {
			input.VersionIdMarker = versionIdMarker
		}

		outputs, err := sb.Client.ListObjectVersions(sb.Context, input)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		// Delete object versions if any exist
		if len(outputs.Versions) > 0 {
			logging.Debugf("Deleting page %d of object versions (%d objects) from bucket %s", pageId, len(outputs.Versions), aws.ToString(bucketName))
			if err := sb.deleteObjectVersions(bucketName, outputs.Versions); err != nil {
				logging.Errorf("Error deleting objects versions for page %d from bucket %s: %s", pageId, aws.ToString(bucketName), err)
				return errors.WithStackTrace(err)
			}
			logging.Debugf("[OK] - deleted page %d of object versions (%d objects) from bucket %s", pageId, len(outputs.Versions), aws.ToString(bucketName))
		}

		// Delete deletion markers if any exist
		if len(outputs.DeleteMarkers) > 0 {
			logging.Debugf("Deleting page %d of deletion markers (%d objects) from bucket %s", pageId, len(outputs.DeleteMarkers), aws.ToString(bucketName))
			if err := sb.deleteDeletionMarkers(bucketName, outputs.DeleteMarkers); err != nil {
				logging.Errorf("Error deleting deletion markers for page %d from bucket %s: %s", pageId, aws.ToString(bucketName), err)
				return errors.WithStackTrace(err)
			}
			logging.Debugf("[OK] - deleted page %d of deletion markers (%d deletion markers) from bucket %s", pageId, len(outputs.DeleteMarkers), aws.ToString(bucketName))
		}

		// Check if there are more pages
		if !aws.ToBool(outputs.IsTruncated) {
			break
		}

		// Set up for next page
		keyMarker = outputs.NextKeyMarker
		versionIdMarker = outputs.NextVersionIdMarker
		pageId++
	}

	paginator := s3.NewListObjectsV2Paginator(sb.Client, &s3.ListObjectsV2Input{
		Bucket:  bucketName,
		MaxKeys: aws.Int32(int32(sb.MaxBatchSize())),
	})

	for paginator.HasMorePages() {

		page, err := paginator.NextPage(sb.Context)
		if err != nil {
			return errors.WithStackTrace(err)
		}

		logging.Debugf("Deleting object page %d (%d objects) from bucket %s", pageId, len(page.Contents), aws.ToString(bucketName))
		if err := sb.deleteObjects(bucketName, page.Contents); err != nil {
			logging.Errorf("Error deleting objects for page %d from bucket %s: %s", pageId, aws.ToString(bucketName), err)
			return err
		}
		pageId++
	}
	return nil
}

// deleteObjects will delete the provided objects (unversioned) from the specified bucket.
func (sb S3Buckets) deleteObjects(bucketName *string, objects []types.Object) error {
	if len(objects) == 0 {
		logging.Debugf("No objects returned in page")
		return nil
	}

	objectIdentifiers := []types.ObjectIdentifier{}
	for _, obj := range objects {
		objectIdentifiers = append(objectIdentifiers, types.ObjectIdentifier{
			Key: obj.Key,
		})
	}
	_, err := sb.Client.DeleteObjects(
		sb.Context,
		&s3.DeleteObjectsInput{
			Bucket: bucketName,
			Delete: &types.Delete{
				Objects: objectIdentifiers,
				Quiet:   aws.Bool(false),
			},
		},
	)
	return err
}

// deleteObjectVersions will delete the provided object versions from the specified bucket.
func (sb S3Buckets) deleteObjectVersions(bucketName *string, objectVersions []types.ObjectVersion) error {
	if len(objectVersions) == 0 {
		logging.Debugf("No object versions returned in page")
		return nil
	}

	objectIdentifiers := []types.ObjectIdentifier{}
	for _, obj := range objectVersions {
		objectIdentifiers = append(objectIdentifiers, types.ObjectIdentifier{
			Key:       obj.Key,
			VersionId: obj.VersionId,
		})
	}
	_, err := sb.Client.DeleteObjects(
		sb.Context,
		&s3.DeleteObjectsInput{
			Bucket: bucketName,
			Delete: &types.Delete{
				Objects: objectIdentifiers,
				Quiet:   aws.Bool(false),
			},
		},
	)
	return err
}

// deleteDeletionMarkers will delete the provided deletion markers from the specified bucket.
func (sb S3Buckets) deleteDeletionMarkers(bucketName *string, objectDelMarkers []types.DeleteMarkerEntry) error {
	if len(objectDelMarkers) == 0 {
		logging.Debugf("No deletion markers returned in page")
		return nil
	}

	objectIdentifiers := []types.ObjectIdentifier{}
	for _, obj := range objectDelMarkers {
		objectIdentifiers = append(objectIdentifiers, types.ObjectIdentifier{
			Key:       obj.Key,
			VersionId: obj.VersionId,
		})
	}
	_, err := sb.Client.DeleteObjects(
		sb.Context,
		&s3.DeleteObjectsInput{
			Bucket: bucketName,
			Delete: &types.Delete{
				Objects: objectIdentifiers,
				Quiet:   aws.Bool(false),
			},
		},
	)
	return err
}

// nukeAllS3BucketObjects batch deletes all objects in an S3 bucket
func (sb S3Buckets) nukeAllS3BucketObjects(bucketName *string) error {
	if sb.MaxBatchSize() < 1 || sb.MaxBatchSize() > 1000 {
		return fmt.Errorf("Invalid batchsize - %d - should be between %d and %d", sb.MaxBatchSize(), 1, 1000)
	}

	logging.Debugf("Emptying bucket %s", aws.ToString(bucketName))
	if err := sb.emptyBucket(bucketName); err != nil {
		return err
	}
	logging.Debugf("[OK] - successfully emptied bucket %s", aws.ToString(bucketName))
	return nil
}

// nukeEmptyS3Bucket deletes an empty S3 bucket
func (sb S3Buckets) nukeEmptyS3Bucket(bucketName *string, verifyBucketDeletion bool) error {

	_, err := sb.Client.DeleteBucket(sb.Context, &s3.DeleteBucketInput{
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
		logging.Debugf("Waiting until bucket (%s) deletion is propagated (attempt %d / %d)", aws.ToString(bucketName), i+1, maxRetries)
		err = waitForBucketDeletion(sb.Context, sb.Client, aws.ToString(bucketName))
		// Exit early if no error
		if err == nil {
			logging.Debug("Successfully detected bucket deletion.")
			return nil
		}
		logging.Debugf("Error waiting for bucket (%s) deletion propagation (attempt %d / %d)", aws.ToString(bucketName), i+1, maxRetries)
		logging.Debugf("Underlying error was: %s", err)
	}
	return err
}

func waitForBucketDeletion(ctx context.Context, client S3API, bucketName string) error {
	waiter := s3.NewBucketNotExistsWaiter(client)

	for i := 0; i < maxRetries; i++ {
		logging.Debugf("Waiting until bucket (%s) deletion is propagated (attempt %d / %d)", bucketName, i+1, maxRetries)

		err := waiter.Wait(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(bucketName),
		}, waitDuration)
		if err == nil {
			logging.Debugf("Successfully detected bucket deletion.")
			return nil
		}
		logging.Debugf("Waiting until bucket erorr (%v)", err)

		if i == maxRetries-1 {
			return fmt.Errorf("failed to confirm bucket deletion after %d attempts: %w", maxRetries, err)
		}

		logging.Debugf("Error waiting for bucket (%s) deletion propagation (attempt %d / %d)", bucketName, i+1, maxRetries)
		logging.Debugf("Underlying error was: %s", err)
	}

	return fmt.Errorf("unexpected error: reached end of retry loop")
}

func (sb S3Buckets) nukeS3BucketPolicy(bucketName *string) error {
	_, err := sb.Client.DeleteBucketPolicy(
		sb.Context,
		&s3.DeleteBucketPolicyInput{
			Bucket: aws.String(*bucketName),
		})
	return err
}

func (sb S3Buckets) nukeS3BucketLifecycle(bucketName *string) error {
	_, err := sb.Client.DeleteBucketLifecycle(
		sb.Context,
		&s3.DeleteBucketLifecycleInput{
			Bucket: aws.String(*bucketName),
		})
	return err
}

func (sb S3Buckets) nukeBucket(bucketName *string) error {
	verifyBucketDeletion := true

	err := sb.nukeAllS3BucketObjects(bucketName)
	if err != nil {
		return err
	}

	err = sb.nukeS3BucketPolicy(bucketName)
	if err != nil {
		return err
	}

	err = sb.nukeS3BucketLifecycle(bucketName)
	if err != nil {
		return err
	}

	err = sb.nukeEmptyS3Bucket(bucketName, verifyBucketDeletion)
	if err != nil {
		return err
	}
	return nil
}

// nukeAllS3Buckets deletes all S3 buckets passed as input
func (sb S3Buckets) nukeAll(bucketNames []*string) (delCount int, err error) {

	if len(bucketNames) == 0 {
		logging.Debugf("No S3 Buckets to nuke in region %s", sb.Region)
		return 0, nil
	}

	totalCount := len(bucketNames)

	logging.Debugf("Deleting - %d S3 Buckets in region %s", totalCount, sb.Region)

	multiErr := new(multierror.Error)

	var deleted []*string
	for bucketIndex := 0; bucketIndex < totalCount; bucketIndex++ {

		bucketName := bucketNames[bucketIndex]
		logging.Debugf("Deleting - %d/%d - Bucket: %s", bucketIndex+1, totalCount, *bucketName)

		err := sb.nukeBucket(bucketName)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(bucketName),
			ResourceType: "S3 Bucket",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] - %d/%d - Bucket: %s - bucket deletion error - %s", bucketIndex+1, totalCount, *bucketName, err)
		} else {
			deleted = append(deleted, bucketName)
			logging.Debugf("[OK] - %d/%d - Bucket: %s - deleted", bucketIndex+1, totalCount, *bucketName)
			delCount++
		}

	}

	logging.Debugf("[OK] - %d Bucket(s) deleted in %s", len(deleted), sb.Region)

	return delCount, multiErr.ErrorOrNil()
}
