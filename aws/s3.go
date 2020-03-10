package aws

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
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

	result, err := svc.GetBucketTagging(input)
	if err != nil {
		return tags, err
	}

	for _, tagSet := range result.TagSet {
		tags = append(tags, map[string]string{"Key": *tagSet.Key, "Value": *tagSet.Value})
	}

	return tags, err
}

// getAllS3Buckets lists and returns a map of per region AWS S3 buckets which were created before excludeAfter
func getAllS3Buckets(session *session.Session, excludeAfter time.Time, bucketNameSubStr string) (map[string][]*string, error) {
	svc := s3.New(session)

	input := &s3.ListBucketsInput{}

	output, err := svc.ListBuckets(input)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var bucketNamesPerRegion = make(map[string][]*string)

OUTER:
	for _, bucket := range output.Buckets {
		logging.Logger.Debugf("Checking - Bucket %s", *bucket.Name)

		if len(bucketNameSubStr) > 0 {
			if !strings.Contains(*bucket.Name, bucketNameSubStr) {
				logging.Logger.Debugf("Skipping - Bucket %s - failed substring filter - %s", *bucket.Name, bucketNameSubStr)
				continue
			}
		}

		if !excludeAfter.After(*bucket.CreationDate) {
			logging.Logger.Debugf("Skipping - Bucket %s - matched CreationDate filter", *bucket.Name)
			continue
		}

		bucketTags, err := getS3BucketTags(svc, bucket.Name)
		if len(bucketTags) > 0 {
			for _, tagSet := range bucketTags {
				if tagSet["Key"] == "cloud-nuke-excluded" && tagSet["Value"] == "true" {
					logging.Logger.Infof("Skipping - Bucket %s - matched tag filter", *bucket.Name)
					continue OUTER
				}
			}
		}

		bucketRegion, err := getS3BucketRegion(svc, bucket.Name)
		if err != nil {
			logging.Logger.Warnf("Skipping - Bucket %s - Failed to get bucket location.", *bucket.Name)
			continue
		}

		if _, ok := bucketNamesPerRegion[bucketRegion]; !ok {
			bucketNamesPerRegion[bucketRegion] = []*string{}
		}

		bucketNamesPerRegion[bucketRegion] = append(bucketNamesPerRegion[bucketRegion], bucket.Name)
	}

	return bucketNamesPerRegion, nil
}

func getS3BucketObjectVersions(svc *s3.S3, bucketName *string, objectKey *string) ([]*string, error) {
	versions := []*string{}

	err := svc.ListObjectVersionsPages(
		&s3.ListObjectVersionsInput{
			Bucket: bucketName,
			Prefix: objectKey,
		},
		func(page *s3.ListObjectVersionsOutput, lastPage bool) (shouldContinue bool) {
			for _, obj := range page.Versions {
				versions = append(versions, obj.VersionId)
			}
			return true
		},
	)
	return versions, err
}

func getS3BucketObjects(svc *s3.S3, bucketName *string, isVersioned bool) ([]*s3.ObjectIdentifier, error) {
	identifiers := []*s3.ObjectIdentifier{}
	hasError := false

	err := svc.ListObjectsV2Pages(
		&s3.ListObjectsV2Input{
			Bucket:  bucketName,
			MaxKeys: aws.Int64(3),
		},

		func(page *s3.ListObjectsV2Output, lastPage bool) (shouldContinue bool) {

			for _, obj := range page.Contents {
				if isVersioned {
					versions, err := getS3BucketObjectVersions(svc, bucketName, obj.Key)
					if err != nil {
						logging.Logger.Warnf("Skipping - Bucket %s - object %s - failed to get version", *bucketName, err.Error())
						hasError = true
						return false
					}

					for _, version := range versions {
						logging.Logger.Debugf("Bucket %s object %s version %s", *bucketName, *obj.Key, *version)
						identifiers = append(identifiers, &s3.ObjectIdentifier{
							Key:       obj.Key,
							VersionId: version,
						})
					}
					continue
				}

				logging.Logger.Debugf("Bucket %s object %s", *bucketName, *obj.Key)

				identifiers = append(identifiers, &s3.ObjectIdentifier{
					Key: obj.Key,
				})
			}
			return true
		},
	)

	if hasError {
		return identifiers, fmt.Errorf("Bucket %s - get object versions failed - check logged errors", *bucketName)
	}

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

	if batchSize <= 0 || batchSize > 1000 {
		batchSize = 1000
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
func nukeAllS3Buckets(session *session.Session, bucketNames []*string, objectBatchSize int) (delCount int, err error) {
	svc := s3.New(session)
	verifyBucketDeletion := true

	if len(bucketNames) == 0 {
		logging.Logger.Infof("No S3 Buckets to nuke in region %s", *session.Config.Region)
		return 0, nil
	}

	totalCount := len(bucketNames)

	logging.Logger.Infof("Deleting - %d S3 Buckets in region %s", totalCount, *session.Config.Region)

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
