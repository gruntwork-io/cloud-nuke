package aws

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// S3GenBucketName generates a test bucket name.
func S3GenBucketName() string {
	return strings.ToLower("cloud-nuke-test-" + util.UniqueID() + util.UniqueID())
}

// S3CreateBucket creates a test bucket and optionally tags and versions it.
func S3CreateBucket(svc *s3.S3, bucketName string, tags []map[string]string, isVersioned bool) error {
	logging.Logger.Debugf("Bucket: %s - creating", bucketName)

	_, err := svc.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return err
	}

	// Add default tag for testing
	var awsTagSet []*s3.Tag

	for _, tagSet := range tags {
		awsTagSet = append(awsTagSet, &s3.Tag{Key: aws.String(tagSet["Key"]), Value: aws.String(tagSet["Value"])})
	}

	if len(awsTagSet) > 0 {
		input := &s3.PutBucketTaggingInput{
			Bucket: aws.String(bucketName),
			Tagging: &s3.Tagging{
				TagSet: awsTagSet,
			},
		}
		_, err = svc.PutBucketTagging(input)
		if err != nil {
			return err
		}
	}

	if isVersioned {
		input := &s3.PutBucketVersioningInput{
			Bucket: aws.String(bucketName),
			VersioningConfiguration: &s3.VersioningConfiguration{
				Status: aws.String("Enabled"),
			},
		}
		_, err = svc.PutBucketVersioning(input)
		if err != nil {
			return err
		}
	}

	return svc.WaitUntilBucketExists(
		&s3.HeadBucketInput{
			Bucket: aws.String(bucketName),
		},
	)
}

// S3BucketAddObject adds an object ot an S3 bucket.
func S3BucketAddObject(awsSession *session.Session, bucketName string, fileName string, fileBody string) error {
	logging.Logger.Debugf("Bucket: %s - adding object: %s - content: %s", bucketName, fileName, fileBody)

	reader := strings.NewReader(fileBody)
	uploader := s3manager.NewUploader(awsSession)

	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileName),
		Body:   reader,
	})
	if err != nil {
		return err
	}
	return nil
}
