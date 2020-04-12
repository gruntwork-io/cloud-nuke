package aws

import (
	"fmt"
	"os"
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
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	logLevel := os.Getenv("LOG_LEVEL")
	if len(logLevel) > 0 {
		parsedLogLevel, err := logrus.ParseLevel(logLevel)
		if err != nil {
			logging.Logger.Errorf("Invalid log level - %s - %s", logLevel, err)
			os.Exit(1)
		}
		logging.Logger.Level = parsedLogLevel
	}
	exitVal := m.Run()
	os.Exit(exitVal)
}

// TestS3Bucket represents a test S3 bucket.
type TestS3Bucket struct {
	name        string
	tags        []map[string]string
	isVersioned bool
}

// SetupInfo has test case setup info.
type SetupInfo struct {
	region     string
	awsSession *session.Session
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

	for _, tagSet := range b.tags {
		awsTagSet = append(awsTagSet, &s3.Tag{Key: aws.String(tagSet["Key"]), Value: aws.String(tagSet["Value"])})
	}

	if len(awsTagSet) > 0 {
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
	uploader := s3manager.NewUploader(s.awsSession)

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

// createSession creates a new session and returns it.
func createSession(region string) (*session.Session, error) {
	if region == "" {
		var err error
		region, err = getRandomRegion()
		if err != nil {
			return nil, err
		}
		logging.Logger.Infof("Creating session in region - %s", region)
	}
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	return session, err
}

// setupNukeTests sets up common operations for nuke S3 tests.
func setupNukeTests(t *testing.T) SetupInfo {
	var s SetupInfo

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	s.region = region

	s.awsSession, err = createSession(region)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	s.svc = s3.New(s.awsSession)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	s.bucketName = genTestBucketName()
	return s
}

func testListS3BucketsWrapper(t *testing.T, bucketTags []map[string]string, batchSize int, shouldMatch bool) {
	s := setupNukeTests(t)

	// Even if we nuke the bucket during our test - this will serve as a test to nuke non-existent bucket
	// + also as our cleanup call

	// Please note that we are passing the same session that was used to create the bucket
	// This is required so that the defer cleanup call always gets the right bucket region
	// to delete
	defer nukeAllS3Buckets(s.awsSession, []*string{aws.String(s.bucketName)}, 1000)

	awsSession, err := createSession("")
	if err != nil {
		assert.Failf(t, "Failed to create random session", errors.WithStackTrace(err).Error())
	}

	targetRegions := []string{s.region}

	// Verify that - before creating bucket - it should not exist
	//
	// Please note that we are not reusing s.awsSession and creating a random session in a region other
	// than the one in which the bucket is created - this is useful to test the sceanrio where the user has
	// AWS_DEFAULT_REGION set to region x but the bucket is in region y.
	bucketNamesPerRegion, err := getAllS3Buckets(awsSession, time.Now().Add(1*time.Hour*-1), targetRegions, s.bucketName, batchSize)
	if batchSize < 0 {
		if err == nil {
			assert.Fail(t, "Did not fail for invalid batch size")
		}
		return
	}

	if err != nil {
		assert.Failf(t, "Failed to list S3 Buckets", errors.WithStackTrace(err).Error())
	} else {
		assert.NotContains(t, bucketNamesPerRegion[s.region], s.bucketName)
	}

	// Create test bucket
	bucket := TestS3Bucket{
		name: s.bucketName,
	}
	if bucketTags != nil && len(bucketTags) > 0 {
		bucket.tags = bucketTags
	}
	err = bucket.create(s.svc)
	if err != nil {
		assert.Failf(t, "Failed to create test bucket", errors.WithStackTrace(err).Error())
	}

	bucketNamesPerRegion, err = getAllS3Buckets(awsSession, time.Now().Add(1*time.Hour), targetRegions, s.bucketName, batchSize)
	if err != nil {
		assert.Failf(t, "Failed to list S3 Buckets", errors.WithStackTrace(err).Error())
	}

	if bucketTags != nil && len(bucketTags) > 0 {
		for _, tag := range bucketTags {
			key := strings.ToLower(tag["Key"])
			value := strings.ToLower(tag["Value"])
			if key == "cloud-nuke-excluded" && value == "true" {
				assert.NotContains(t, bucketNamesPerRegion[s.region], aws.String(s.bucketName))
			}
		}
	}

	if shouldMatch {
		assert.Contains(t, bucketNamesPerRegion[s.region], aws.String(s.bucketName))
	}
}

func TestList_EmptyS3Bucket_NoTags(t *testing.T) {
	testListS3BucketsWrapper(t, []map[string]string{}, 10, true)
}

func TestList_EmptyS3Bucket_WithoutFilterTag(t *testing.T) {
	testListS3BucketsWrapper(t, []map[string]string{
		{
			"Key":   "testKey",
			"Value": "testValue",
		},
	}, 10, true)
}

func TestList_EmptyS3Bucket_WithFilterTag(t *testing.T) {
	// Test single filter key
	testListS3BucketsWrapper(t, []map[string]string{
		{
			"Key":   AwsResourceExclusionTagKey,
			"Value": "true",
		},
	}, 10, false)

	// Test filter key with other keys + validate multi case filter key
	testListS3BucketsWrapper(t, []map[string]string{
		{
			"Key":   "test-key-1",
			"Value": "test-value-1",
		},
		{
			"Key":   "test-key-2",
			"Value": "test-value-2",
		},
		{
			"Key":   strings.ToTitle(AwsResourceExclusionTagKey),
			"Value": "TruE",
		},
	}, 10, false)
}

func TestList_EmptyS3Bucket_InvalidBatchSize(t *testing.T) {
	testListS3BucketsWrapper(t, nil, -1, false)
}

func TestNuke_EmptyS3Bucket(t *testing.T) {
	s := setupNukeTests(t)

	// Create test bucket
	bucket := TestS3Bucket{
		name: s.bucketName,
	}

	err := bucket.create(s.svc)
	if err != nil {
		assert.Failf(t, "Failed to create test bucket", errors.WithStackTrace(err).Error())
	}

	// Please note that we are passing the same session that was used to create the bucket
	// This is required so that the defer cleanup call always gets the right bucket region
	// to delete
	defer nukeAllS3Buckets(s.awsSession, []*string{aws.String(s.bucketName)}, 1000)

	// Create separate session object as bucket region may not be equal to session region
	awsSession, err := createSession("")
	if err != nil {
		assert.Failf(t, "Failed to create random session", errors.WithStackTrace(err).Error())
	}

	// Nuke the test bucket
	_, err = nukeAllS3Buckets(s.awsSession, []*string{aws.String(s.bucketName)}, 1000)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	// Verify that - after nuking test bucket - it should not exist
	bucketNamesPerRegion, err := getAllS3Buckets(awsSession, time.Now().Add(1*time.Hour), []string{s.region}, s.bucketName, 100)
	if err != nil {
		assert.Failf(t, "Failed to list S3 Buckets", errors.WithStackTrace(err).Error())
	} else {
		assert.NotContains(t, bucketNamesPerRegion[s.region], aws.String(s.bucketName))
	}
}

func testNukeS3BucketWrapper(t *testing.T, args *TestNukeS3BucketArgs, shouldNuke bool) {
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

	awsSession, err := createSession("")
	if err != nil {
		assert.Failf(t, "Failed to create random session", errors.WithStackTrace(err).Error())
	}

	defer nukeAllS3Buckets(s.awsSession, []*string{aws.String(s.bucketName)}, 1000)

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
	delCount, err := nukeAllS3Buckets(s.awsSession, []*string{aws.String(s.bucketName)}, args.objectBatchsize)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	// If we should not nuke the bucket then deleted bucket count should be 0.
	if !shouldNuke {
		if delCount != 0 {
			assert.Failf(t, "Should not nuke but got delCount > 0", "delCount: %d", delCount)
		}
		return
	}

	// Verify that - after nuking test bucket - it should not exist
	bucketNamesPerRegion, err := getAllS3Buckets(awsSession, time.Now().Add(1*time.Hour), []string{s.region}, s.bucketName, 100)
	if err != nil {
		assert.Failf(t, "Failed to list S3 Buckets", errors.WithStackTrace(err).Error())
	} else {
		assert.NotContains(t, bucketNamesPerRegion[s.region], aws.String(s.bucketName))
	}
}

func TestNuke_S3Bucket_WithoutVersioning_AllObjects(t *testing.T) {
	testNukeS3BucketWrapper(t, &TestNukeS3BucketArgs{
		isVersioned:       false,
		checkDeleteMarker: false,
		objectCount:       10,
		objectBatchsize:   1000,
	}, true)
}

func TestNuke_S3Bucket_WithoutVersioning_BatchObjects(t *testing.T) {
	testNukeS3BucketWrapper(t, &TestNukeS3BucketArgs{
		isVersioned:       false,
		checkDeleteMarker: false,
		objectCount:       10,
		objectBatchsize:   2,
	}, true)
}

func TestNuke_S3Bucket_WithoutVersioning_BatchObjects_InvalidBatchSize(t *testing.T) {
	testNukeS3BucketWrapper(t, &TestNukeS3BucketArgs{
		isVersioned:       false,
		checkDeleteMarker: false,
		objectCount:       2,
		objectBatchsize:   1001,
	}, false)

	testNukeS3BucketWrapper(t, &TestNukeS3BucketArgs{
		isVersioned:       false,
		checkDeleteMarker: false,
		objectCount:       2,
		objectBatchsize:   0,
	}, false)
}

func TestNuke_S3Bucket_WithVersioning_AllObjects(t *testing.T) {
	testNukeS3BucketWrapper(t, &TestNukeS3BucketArgs{
		isVersioned:       true,
		checkDeleteMarker: false,
		objectCount:       10,
		objectBatchsize:   1000,
	}, true)
}

func TestNuke_S3Bucket_WithVersioning_BatchObjects(t *testing.T) {
	testNukeS3BucketWrapper(t, &TestNukeS3BucketArgs{
		isVersioned:       true,
		checkDeleteMarker: false,
		objectCount:       10,
		objectBatchsize:   3,
	}, true)
}

func TestNuke_S3Bucket_WithVersioning_DeleteMarker(t *testing.T) {
	testNukeS3BucketWrapper(t, &TestNukeS3BucketArgs{
		isVersioned:       true,
		checkDeleteMarker: true,
		objectCount:       10,
		objectBatchsize:   1000,
	}, true)
}
