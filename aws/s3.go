package aws

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// getS3BucketRegion returns S3 Bucket region.
func getS3BucketRegion(svc *s3.S3, bucketName *string) (string, error) {
	input := &s3.GetBucketLocationInput{
		Bucket: bucketName,
	}

	result, err := svc.GetBucketLocation(input)
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
func getS3BucketTags(svc *s3.S3, bucketName *string) ([]map[string]string, error) {
	input := &s3.GetBucketTaggingInput{
		Bucket: bucketName,
	}

	tags := []map[string]string{}

	// Please note that svc argument should be created from a session object which is
	// in the same region as the bucket or GetBucketTagging will fail.
	result, err := svc.GetBucketTagging(input)
	if err != nil {
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
	Name           *string
	CreationDate   *time.Time
	Region         *string
	Tags           []map[string]string
	Error          error
	InTargetRegion bool
}

// getAllS3Buckets lists and returns a map of per region AWS S3 buckets which were created before excludeAfter
func getAllS3Buckets(awsSession *session.Session, excludeAfter time.Time,
	targetRegions []string, batchSize int, bucketNameSubStr string) (map[string][]*string, error) {

	svc := s3.New(awsSession)

	input := &s3.ListBucketsInput{}

	output, err := svc.ListBuckets(input)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var bucketNamesPerRegion = make(map[string][]*string)
	var regionClients = make(map[string]*s3.S3)

	for _, region := range targetRegions {
		logging.Logger.Debugf("S3 - creating session - region %s", region)
		awsSession, err := session.NewSession(&awsgo.Config{
			Region: awsgo.String(region)},
		)
		if err != nil {
			return bucketNamesPerRegion, errors.WithStackTrace(err)
		}
		regionClients[region] = s3.New(awsSession)
	}

	totalBuckets := len(output.Buckets)
	if totalBuckets == 0 {
		return bucketNamesPerRegion, nil
	}

	if batchSize <= 0 {
		return nil, fmt.Errorf("Invalid batchsize - %d - should be > 0", batchSize)
	}

	totalBatches := totalBuckets / batchSize
	if totalBuckets%batchSize != 0 {
		totalBatches = totalBatches + 1
	}
	batchCount := 1

	// Batch the get operation
	for i := 0; i < totalBuckets; i += batchSize {
		j := i + batchSize
		if j > totalBuckets {
			j = totalBuckets
		}

		logging.Logger.Infof("Getting - %d-%d buckets of batch %d/%d", i+1, j, batchCount, totalBatches)

		targetBuckets := output.Buckets[i:j]

		var bucketCh = make(chan *S3Bucket, len(targetBuckets))
		var wg sync.WaitGroup

		for _, bucket := range targetBuckets {
			if len(bucketNameSubStr) > 0 && !strings.Contains(*bucket.Name, bucketNameSubStr) {
				logging.Logger.Debugf("Skipping - Bucket %s - failed substring filter - %s", *bucket.Name, bucketNameSubStr)
				continue
			}

			wg.Add(1)
			go func(bucket *s3.Bucket) {
				defer wg.Done()
				if err := getBucketInfo(svc, bucket, regionClients, bucketCh); err != nil {
					logging.Logger.Warnf("Skipping - Bucket %s - Failed to get info - %s", *bucket.Name, err)
				}
			}(bucket)
		}

		go func() {
			wg.Wait()
			close(bucketCh)
		}()

		for b := range bucketCh {
			if !b.InTargetRegion {
				logging.Logger.Debugf("Skipping - Bucket %s - region - %s - not in target regions", *b.Name, *b.Region)
				continue
			}

			if !excludeAfter.After(*b.CreationDate) {
				logging.Logger.Debugf("Skipping - Bucket %s - matched CreationDate filter", *b.Name)
				continue
			}

			if !hasValidTags(b.Tags) {
				logging.Logger.Debugf("Skipping - Bucket %s - matched tag filter", *b.Name)
				continue
			}

			if _, ok := bucketNamesPerRegion[*b.Region]; !ok {
				bucketNamesPerRegion[*b.Region] = []*string{}
			}

			bucketNamesPerRegion[*b.Region] = append(bucketNamesPerRegion[*b.Region], b.Name)
		}

		batchCount++
	}

	return bucketNamesPerRegion, nil
}

func getBucketInfo(svc *s3.S3, bucket *s3.Bucket, regionClients map[string]*s3.S3, bucketCh chan<- *S3Bucket) error {
	var b S3Bucket
	b.Name = bucket.Name
	b.CreationDate = bucket.CreationDate

	bucketRegion, err := getS3BucketRegion(svc, b.Name)
	if err != nil {
		b.Error = err
		return err
	}
	b.Region = &bucketRegion

	for region := range regionClients {
		if region == *b.Region {
			b.InTargetRegion = true
			break
		}
	}

	// Bucket not in any of the target regions - skip checking tags
	if !b.InTargetRegion {
		bucketCh <- &b
		return nil
	}

	bucketTags, err := getS3BucketTags(regionClients[*b.Region], b.Name)
	if err != nil {
		b.Error = err
		return err
	}
	b.Tags = bucketTags

	bucketCh <- &b
	return nil
}

func getS3BucketObjects(svc *s3.S3, bucketName *string, isVersioned bool) ([]*s3.ObjectIdentifier, error) {
	identifiers := []*s3.ObjectIdentifier{}

	// Handle versioned buckets.
	if isVersioned {
		err := svc.ListObjectVersionsPages(
			&s3.ListObjectVersionsInput{
				Bucket: bucketName,
			},
			func(page *s3.ListObjectVersionsOutput, lastPage bool) (shouldContinue bool) {
				for _, obj := range page.Versions {
					logging.Logger.Debugf("Bucket %s object %s version %s", *bucketName, *obj.Key, *obj.VersionId)
					identifiers = append(identifiers, &s3.ObjectIdentifier{
						Key:       obj.Key,
						VersionId: obj.VersionId,
					})
				}
				for _, obj := range page.DeleteMarkers {
					logging.Logger.Debugf("Bucket %s object %s DeleteMarker %s", *bucketName, *obj.Key, *obj.VersionId)
					identifiers = append(identifiers, &s3.ObjectIdentifier{
						Key:       obj.Key,
						VersionId: obj.VersionId,
					})
				}
				return true
			},
		)
		return identifiers, err
	}

	// Handle non versioned buckets.
	err := svc.ListObjectsV2Pages(
		&s3.ListObjectsV2Input{
			Bucket: bucketName,
		},
		func(page *s3.ListObjectsV2Output, lastPage bool) (shouldContinue bool) {
			for _, obj := range page.Contents {
				logging.Logger.Debugf("Bucket %s object %s", *bucketName, *obj.Key)
				identifiers = append(identifiers, &s3.ObjectIdentifier{
					Key: obj.Key,
				})
			}
			return true
		},
	)

	return identifiers, err
}

func nukeAllS3BucketObjects(svc *s3.S3, bucketName *string, batchSize int) error {
	versioningResult, err := svc.GetBucketVersioning(&s3.GetBucketVersioningInput{
		Bucket: bucketName,
	})
	if err != nil {
		return err
	}

	isVersioned := versioningResult.Status != nil && *versioningResult.Status == "Enabled"

	objects, err := getS3BucketObjects(svc, bucketName, isVersioned)
	if err != nil {
		return err
	}

	totalObjects := len(objects)

	if totalObjects == 0 {
		logging.Logger.Infof("Bucket: %s - empty - skipping object deletion", *bucketName)
		return nil
	}

	if batchSize < 1 || batchSize > 1000 {
		return fmt.Errorf("Invalid batchsize - %d - should be between %d and %d", batchSize, 1, 1000)
	}

	logging.Logger.Infof("Deleting - Bucket: %s - objects: %d", *bucketName, totalObjects)

	totalBatches := totalObjects / batchSize
	if totalObjects%batchSize != 0 {
		totalBatches = totalBatches + 1
	}
	batchCount := 1

	// Batch the delete operation
	for i := 0; i < len(objects); i += batchSize {
		j := i + batchSize
		if j > len(objects) {
			j = len(objects)
		}

		logging.Logger.Debugf("Deleting - %d-%d objects of batch %d/%d - Bucket: %s", i+1, j, batchCount, totalBatches, *bucketName)

		delObjects := objects[i:j]
		_, err = svc.DeleteObjects(
			&s3.DeleteObjectsInput{
				Bucket: bucketName,
				Delete: &s3.Delete{
					Objects: delObjects,
					Quiet:   aws.Bool(false),
				},
			},
		)
		if err != nil {
			return err
		}

		logging.Logger.Infof("[OK] - %d-%d objects of batch %d/%d - Bucket: %s - deleted", i+1, j, batchCount, totalBatches, *bucketName)

		batchCount++
	}

	return nil
}

func nukeEmptyS3Bucket(svc *s3.S3, bucketName *string, verifyBucketDeletion bool) error {
	_, err := svc.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: bucketName,
	})
	if err != nil {
		return err
	}

	if !verifyBucketDeletion {
		return err
	}

	err = svc.WaitUntilBucketNotExists(&s3.HeadBucketInput{
		Bucket: bucketName,
	})
	return err
}

// nukeAllS3Buckets deletes all S3 buckets passed as input
func nukeAllS3Buckets(awsSession *session.Session, bucketNames []*string, objectBatchSize int) (delCount int, err error) {
	svc := s3.New(awsSession)
	verifyBucketDeletion := true

	if len(bucketNames) == 0 {
		logging.Logger.Infof("No S3 Buckets to nuke in region %s", *awsSession.Config.Region)
		return 0, nil
	}

	totalCount := len(bucketNames)

	logging.Logger.Infof("Deleting - %d S3 Buckets in region %s", totalCount, *awsSession.Config.Region)

	for bucketIndex := 0; bucketIndex < totalCount; bucketIndex++ {

		bucketName := bucketNames[bucketIndex]
		logging.Logger.Debugf("Deleting - %d/%d - Bucket: %s", bucketIndex+1, totalCount, *bucketName)

		err = nukeAllS3BucketObjects(svc, bucketName, objectBatchSize)
		if err != nil {
			logging.Logger.Errorf("[Failed] - %d/%d - Bucket: %s - object deletion error - %s", bucketIndex+1, totalCount, *bucketName, err)
			continue
		}

		err = nukeEmptyS3Bucket(svc, bucketName, verifyBucketDeletion)
		if err != nil {
			logging.Logger.Errorf("[Failed] - %d/%d - Bucket: %s - bucket deletion error - %s", bucketIndex+1, totalCount, *bucketName, err)
			continue
		}

		logging.Logger.Infof("[OK] - %d/%d - Bucket: %s - deleted", bucketIndex+1, totalCount, *bucketName)
		delCount++
	}

	return delCount, nil
}
