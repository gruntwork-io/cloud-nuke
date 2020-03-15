package aws

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

// TestS3Bucket represents a test S3 bucket.
type TestS3Bucket struct {
	name        string
	tags        []map[string]string
	isVersioned bool
}

// SetupInfo has test case setup info.
type SetupInfo struct {
	region     string
	session    *session.Session
	svc        *s3.S3
	bucketName string
}

// TestNukeS3BucketArgs represents arguments for testNukeS3BucketWrapper
type TestNukeS3BucketArgs struct {
	isVersioned       bool
	checkDeleteMarker bool
	objectCount       int
	objectBatchsize   int
}

// genTestsBucketName generates a test bucket name.
func genTestBucketName() string {
	// Call UniqueID twice because even if the nth test tries to reuse a name of the first test
	// AWS S3 deletion operation might be in progress after nuking.
	return strings.ToLower("cloud-nuke-test-" + util.UniqueID() + util.UniqueID())
}

// TestS3Bucket create creates a test bucket
func (b TestS3Bucket) create(svc *s3.S3) error {
	logging.Logger.Infof("Bucket: %s - creating", b.name)

	_, err := svc.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(b.name),
	})
	if err != nil {
		return err
	}

	// Add default tag for testing
	var awsTagSet []*s3.Tag
	awsTagSet = append(awsTagSet, &s3.Tag{Key: aws.String("cloud-nuke-test"), Value: aws.String("true")})

	for _, tagSet := range b.tags {
		awsTagSet = append(awsTagSet, &s3.Tag{Key: aws.String(tagSet["Key"]), Value: aws.String(tagSet["Value"])})
	}

	input := &s3.PutBucketTaggingInput{
		Bucket: aws.String(b.name),
		Tagging: &s3.Tagging{
			TagSet: awsTagSet,
		},
	}
	_, err = svc.PutBucketTagging(input)
	if err != nil {
		return err
	}

	if b.isVersioned {
		input := &s3.PutBucketVersioningInput{
			Bucket: aws.String(b.name),
			VersioningConfiguration: &s3.VersioningConfiguration{
				Status: aws.String("Enabled"),
			},
		}
		_, err = svc.PutBucketVersioning(input)
		if err != nil {
			return err
		}
	}

	err = svc.WaitUntilBucketExists(
		&s3.HeadBucketInput{
			Bucket: aws.String(b.name),
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func (b TestS3Bucket) addObject(s SetupInfo, fileName string, fileBody string) error {
	logging.Logger.Infof("Bucket: %s - adding object: %s - content: %s", b.name, fileName, fileBody)

	reader := strings.NewReader(fileBody)
	uploader := s3manager.NewUploader(s.session)

	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(b.name),
		Key:    aws.String(fileName),
		Body:   reader,
	})
	if err != nil {
		return err
	}

	return nil
}

// setupNukeTests sets up common operations for nuke S3 tests.
func setupNukeTests(t *testing.T) SetupInfo {
	var s SetupInfo

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	s.region = region

	s.session, err = session.NewSession(&awsgo.Config{
		Region: awsgo.String(s.region)},
	)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	s.svc = s3.New(s.session)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	s.bucketName = genTestBucketName()
	return s
}

func testListS3BucketsWrapper(t *testing.T, bucketTags []map[string]string) {
	s := setupNukeTests(t)

	// Even if we nuke the bucket during our test - this will serve as a test to nuke non-existent bucket
	// + also as our cleanup call
	defer nukeAllS3Buckets(s.session, []*string{aws.String(s.bucketName)}, 1000)

	// Verify that - before creating bucket - it should not exist
	bucketNamesPerRegion, err := getAllS3Buckets(s.session, time.Now().Add(1*time.Hour*-1), s.bucketName)
	if err != nil {
		assert.Failf(t, "Failed to list S3 Buckets", errors.WithStackTrace(err).Error())
	} else {
		assert.NotContains(t, bucketNamesPerRegion[s.region], s.bucketName)
	}

	// Create test bucket
	bucket := TestS3Bucket{
		name: s.bucketName,
	}
	if len(bucketTags) > 0 {
		bucket.tags = bucketTags
	}
	err = bucket.create(s.svc)
	if err != nil {
		assert.Failf(t, "Failed to create test bucket", errors.WithStackTrace(err).Error())
	}

	bucketNamesPerRegion, err = getAllS3Buckets(s.session, time.Now().Add(1*time.Hour), s.bucketName)
	if err != nil {
		assert.Failf(t, "Failed to list S3 Buckets", errors.WithStackTrace(err).Error())
	}

	if len(bucketTags) > 0 {
		if bucketTags[0]["Key"] == "cloud-nuke-excluded" && bucketTags[0]["Value"] == "true" {
			assert.NotContains(t, bucketNamesPerRegion[s.region], aws.String(s.bucketName))
			return
		}
	}
	assert.Contains(t, bucketNamesPerRegion[s.region], aws.String(s.bucketName))
}

func TestListEmptyS3BucketWithoutFilterTag(t *testing.T) {
	testListS3BucketsWrapper(t, []map[string]string{
		{
			"Key":   "testKey",
			"Value": "testValue",
		},
	})
}

func TestListEmptyS3BucketWithFilterTag(t *testing.T) {
	testListS3BucketsWrapper(t, []map[string]string{
		{
			"Key":   "cloud-nuke-excluded",
			"Value": "true",
		},
	})
}

func TestNukeEmptyS3Bucket(t *testing.T) {
	s := setupNukeTests(t)

	// Create test bucket
	bucket := TestS3Bucket{
		name: s.bucketName,
	}

	err := bucket.create(s.svc)
	if err != nil {
		assert.Failf(t, "Failed to create test bucket", errors.WithStackTrace(err).Error())
	}

	defer nukeAllS3Buckets(s.session, []*string{aws.String(s.bucketName)}, 1000)

	// Nuke the test bucket
	_, err = nukeAllS3Buckets(s.session, []*string{aws.String(s.bucketName)}, 1000)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	// Verify that - after nuking test bucket - it should not exist
	bucketNamesPerRegion, err := getAllS3Buckets(s.session, time.Now().Add(1*time.Hour), s.bucketName)
	if err != nil {
		assert.Failf(t, "Failed to list S3 Buckets", errors.WithStackTrace(err).Error())
	} else {
		assert.NotContains(t, bucketNamesPerRegion[s.region], aws.String(s.bucketName))
	}
}

func testNukeS3BucketWrapper(t *testing.T, args *TestNukeS3BucketArgs) {
	s := setupNukeTests(t)

	// Create test bucket
	bucket := TestS3Bucket{
		name:        s.bucketName,
		isVersioned: args.isVersioned,
	}
	err := bucket.create(s.svc)
	if err != nil {
		assert.Failf(t, "Failed to create test bucket", errors.WithStackTrace(err).Error())
	}

	defer nukeAllS3Buckets(s.session, []*string{aws.String(s.bucketName)}, 1000)

	objectVersions := 1
	if args.isVersioned {
		objectVersions = 3
	}

	// Add two more versions of the same file
	for i := 0; i < objectVersions; i++ {
		for j := 0; j < args.objectCount; j++ {
			fileName := fmt.Sprintf("l1/l2/l3/f%d.txt", j)
			fileBody := fmt.Sprintf("%d-%d", i, j)
			err := bucket.addObject(s, fileName, fileBody)
			if err != nil {
				assert.Failf(t, "Failed to add object to test bucket", errors.WithStackTrace(err).Error())
			}
		}
	}

	// Do a simple delete to create DeleteMarker object
	if args.checkDeleteMarker {
		targetObject := "l1/l2/l3/f0.txt"
		logging.Logger.Infof("Bucket: %s - doing simple delete on object: %s", s.bucketName, targetObject)

		_, err = s.svc.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(s.bucketName),
			Key:    aws.String("l1/l2/l3/f0.txt"),
		})
		if err != nil {
			assert.Fail(t, errors.WithStackTrace(err).Error())
		}
	}

	// Nuke the test bucket
	_, err = nukeAllS3Buckets(s.session, []*string{aws.String(s.bucketName)}, args.objectBatchsize)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	// Verify that - after nuking test bucket - it should not exist
	bucketNamesPerRegion, err := getAllS3Buckets(s.session, time.Now().Add(1*time.Hour), s.bucketName)
	if err != nil {
		assert.Failf(t, "Failed to list S3 Buckets", errors.WithStackTrace(err).Error())
	} else {
		assert.NotContains(t, bucketNamesPerRegion[s.region], aws.String(s.bucketName))
	}
}

func TestNukeS3BucketWithoutVersioningAllObjects(t *testing.T) {
	// testNukeS3BucketWrapper(t, false, 10, 1000)
	testNukeS3BucketWrapper(t, &TestNukeS3BucketArgs{
		isVersioned:       false,
		checkDeleteMarker: false,
		objectCount:       10,
		objectBatchsize:   1000,
	})
}

func TestNukeS3BucketWithoutVersioningBatchObjects(t *testing.T) {
	testNukeS3BucketWrapper(t, &TestNukeS3BucketArgs{
		isVersioned:       false,
		checkDeleteMarker: false,
		objectCount:       10,
		objectBatchsize:   2,
	})
}

func TestNukeS3BucketWithVersioningAllObjects(t *testing.T) {
	testNukeS3BucketWrapper(t, &TestNukeS3BucketArgs{
		isVersioned:       true,
		checkDeleteMarker: false,
		objectCount:       10,
		objectBatchsize:   1000,
	})
}

func TestNukeS3BucketWithVersioningBatchObjects(t *testing.T) {
	testNukeS3BucketWrapper(t, &TestNukeS3BucketArgs{
		isVersioned:       true,
		checkDeleteMarker: false,
		objectCount:       10,
		objectBatchsize:   3,
	})
}

func TestNukeS3BucketWithVersioningDeleteMarker(t *testing.T) {
	testNukeS3BucketWrapper(t, &TestNukeS3BucketArgs{
		isVersioned:       true,
		checkDeleteMarker: true,
		objectCount:       10,
		objectBatchsize:   1000,
	})
}
