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

// genTestsBucketName generates a test bucket name.
func genTestBucketName() string {
	return strings.ToLower("cloud-nuke-test-" + util.UniqueID() + util.UniqueID())
}

// createSession creates a new session and returns it.
func createSession(region string) (*session.Session, error) {
	if region == "" {
		var err error
		region, err = getRandomRegion()
		if err != nil {
			return nil, err
		}
		logging.Logger.Debugf("Creating session in region - %s", region)
	}
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	return session, err
}

// AWSParams has AWS params info.
type AWSParams struct {
	region     string
	awsSession *session.Session
	svc        *s3.S3
}

// newAWSParams sets up common operations for nuke S3 tests.
func newAWSParams() (AWSParams, error) {
	var params AWSParams

	region, err := getRandomRegion()
	if err != nil {
		return params, err
	}
	params.region = region

	params.awsSession, err = createSession(region)
	if err != nil {
		return params, err
	}

	params.svc = s3.New(params.awsSession)
	if err != nil {
		return params, err
	}

	return params, nil
}

// TestS3Bucket represents a test S3 bucket
type TestS3Bucket struct {
	name        string
	tags        []map[string]string
	isVersioned bool
}

// TestS3Bucket create creates a test bucket
func (b TestS3Bucket) create(svc *s3.S3) error {
	logging.Logger.Debugf("Bucket: %s - creating", b.name)

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

// addObject adds an object to S3 bucket
func (b TestS3Bucket) addObject(awsParams AWSParams, fileName string, fileBody string) error {
	logging.Logger.Debugf("Bucket: %s - adding object: %s - content: %s", b.name, fileName, fileBody)

	reader := strings.NewReader(fileBody)
	uploader := s3manager.NewUploader(awsParams.awsSession)

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

// TestListS3BucketArgs represents arguments for TestListS3Bucket
type TestListS3BucketArgs struct {
	bucketTags  []map[string]string
	batchSize   int
	shouldError bool
	shouldMatch bool
}

// genTestList3Bucket - generates the test function for TestNukeS3Bucket
func genTestListS3Bucket(args TestListS3BucketArgs) (func(t *testing.T), error) {
	awsParams, err := newAWSParams()

	if err != nil {
		return nil, fmt.Errorf("Failed to setup AWS params - %s", err.Error())
	}

	bucket := TestS3Bucket{
		name: genTestBucketName(),
	}

	awsSession, err := createSession("")
	if err != nil {
		return nil, fmt.Errorf("Failed to create random session - %s", err.Error())
	}

	targetRegions := []string{awsParams.region}

	return func(t *testing.T) {
		t.Parallel()

		// Please note that we are passing the same session that was used to create the bucket
		// This is required so that the defer cleanup call always gets the right bucket region
		// to delete
		defer nukeAllS3Buckets(awsParams.awsSession, []*string{aws.String(bucket.name)}, 1000)

		// Verify that - before creating bucket - it should not exist
		//
		// Please note that we are not reusing awsParams.awsSession and creating a random session in a region other
		// than the one in which the bucket is created - this is useful to test the scenario where the user has
		// AWS_DEFAULT_REGION set to region x but the bucket is in region y.
		bucketNamesPerRegion, err := getAllS3Buckets(awsSession, time.Now().Add(1*time.Hour*-1), targetRegions, bucket.name, args.batchSize)
		if args.shouldError {
			if err == nil {
				assert.Fail(t, "Did not fail for invalid batch size")
			}
			logging.Logger.Debugf("SUCCESS: Did not list buckets due to invalid batch size - %s", bucket.name)
			return
		}

		if err != nil {
			assert.Failf(t, "Failed to list S3 Buckets", errors.WithStackTrace(err).Error())
		}

		// Validate test bucket does not exist before creation
		assert.NotContains(t, bucketNamesPerRegion[awsParams.region], aws.String(bucket.name))

		// Create test bucket
		if args.bucketTags != nil && len(args.bucketTags) > 0 {
			bucket.tags = args.bucketTags
		}
		err = bucket.create(awsParams.svc)
		if err != nil {
			assert.Failf(t, "Failed to create test bucket", errors.WithStackTrace(err).Error())
		}

		bucketNamesPerRegion, err = getAllS3Buckets(awsSession, time.Now().Add(1*time.Hour), targetRegions, bucket.name, args.batchSize)
		if err != nil {
			assert.Failf(t, "Failed to list S3 Buckets", errors.WithStackTrace(err).Error())
		}

		if args.shouldMatch {
			assert.Contains(t, bucketNamesPerRegion[awsParams.region], aws.String(bucket.name))
			logging.Logger.Debugf("SUCCESS: Matched bucket - %s", bucket.name)
		} else {
			assert.NotContains(t, bucketNamesPerRegion[awsParams.region], aws.String(bucket.name))
			logging.Logger.Debugf("SUCCESS: Did not match bucket - %s", bucket.name)
		}
	}, nil
}

// TestListS3Bucket tests listing S3 bucket operation
func TestListS3Bucket(t *testing.T) {
	t.Parallel()

	var testCases = []struct {
		name string
		args TestListS3BucketArgs
	}{
		{
			"NoTags",
			TestListS3BucketArgs{
				bucketTags:  []map[string]string{},
				batchSize:   10,
				shouldMatch: true,
				shouldError: false,
			},
		},
		{
			"WithoutFilterTag",
			TestListS3BucketArgs{
				bucketTags: []map[string]string{
					{"Key": "testKey", "Value": "testValue"},
				},
				batchSize:   10,
				shouldMatch: true,
				shouldError: false,
			},
		},
		{
			"WithFilterTag",
			TestListS3BucketArgs{
				bucketTags: []map[string]string{
					{"Key": AwsResourceExclusionTagKey, "Value": "true"},
				},
				batchSize:   10,
				shouldMatch: false,
				shouldError: false,
			},
		},
		{
			"MultiCaseFilterTag",
			TestListS3BucketArgs{
				bucketTags: []map[string]string{
					{"Key": "test-key-1", "Value": "test-value-1"},
					{"Key": "test-key-2", "Value": "test-value-2"},
					{"Key": strings.ToTitle(AwsResourceExclusionTagKey), "Value": "TrUe"},
				},
				batchSize:   10,
				shouldMatch: false,
				shouldError: false,
			},
		},
		{
			"InvalidBatchSize",
			TestListS3BucketArgs{
				bucketTags:  nil,
				batchSize:   -1,
				shouldMatch: false,
				shouldError: true,
			},
		},
	}
	for _, tc := range testCases {
		testFunc, err := genTestListS3Bucket(tc.args)
		if err != nil {
			assert.Fail(t, errors.WithStackTrace(err).Error())
		}
		t.Run(tc.name, testFunc)
	}
}

// TestNukeS3BucketArgs represents arguments for TestNukeS3Bucket
type TestNukeS3BucketArgs struct {
	isVersioned       bool
	checkDeleteMarker bool
	objectCount       int
	objectBatchsize   int
	shouldNuke        bool
}

// genTestNukeS3Bucket - generates the test function for TestNukeS3Bucket
func genTestNukeS3Bucket(args TestNukeS3BucketArgs) (func(t *testing.T), error) {
	awsParams, err := newAWSParams()
	if err != nil {
		return nil, fmt.Errorf("Failed to setup AWS params - %s", err.Error())
	}

	// Create test bucket
	bucket := TestS3Bucket{
		name:        genTestBucketName(),
		isVersioned: args.isVersioned,
	}
	err = bucket.create(awsParams.svc)
	if err != nil {
		return nil, fmt.Errorf("Failed to create test bucket - %s", err.Error())
	}

	awsSession, err := createSession("")
	if err != nil {
		return nil, fmt.Errorf("Failed to create random session - %s", err.Error())
	}

	if args.objectCount > 0 {
		objectVersions := 1
		if args.isVersioned {
			objectVersions = 3
		}

		// Add two more versions of the same file
		for i := 0; i < objectVersions; i++ {
			for j := 0; j < args.objectCount; j++ {
				fileName := fmt.Sprintf("l1/l2/l3/f%d.txt", j)
				fileBody := fmt.Sprintf("%d-%d", i, j)
				err := bucket.addObject(awsParams, fileName, fileBody)
				if err != nil {
					return nil, fmt.Errorf("Failed to add object to test bucket - %s", err.Error())
				}
			}
		}

		// Do a simple delete to create DeleteMarker object
		if args.checkDeleteMarker {
			targetObject := "l1/l2/l3/f0.txt"
			logging.Logger.Debugf("Bucket: %s - doing simple delete on object: %s", bucket.name, targetObject)

			_, err = awsParams.svc.DeleteObject(&s3.DeleteObjectInput{
				Bucket: aws.String(bucket.name),
				Key:    aws.String("l1/l2/l3/f0.txt"),
			})
			if err != nil {
				return nil, fmt.Errorf("Failed to create delete marker - %s", err.Error())
			}
		}
	}

	return func(t *testing.T) {
		t.Parallel()
		defer nukeAllS3Buckets(awsParams.awsSession, []*string{aws.String(bucket.name)}, 1000)

		// Nuke the test bucket
		delCount, err := nukeAllS3Buckets(awsParams.awsSession, []*string{aws.String(bucket.name)}, args.objectBatchsize)
		if err != nil {
			assert.Fail(t, errors.WithStackTrace(err).Error())
		}

		// If we should not nuke the bucket then deleted bucket count should be 0
		if !args.shouldNuke {
			if delCount > 0 {
				assert.Failf(t, "Should not nuke but got delCount > 0", "delCount: %d", delCount)
			}
			logging.Logger.Debugf("SUCCESS: Did not nuke bucket - %s", bucket.name)
			return
		}

		// Verify that - after nuking test bucket - it should not exist
		bucketNamesPerRegion, err := getAllS3Buckets(awsSession, time.Now().Add(1*time.Hour), []string{awsParams.region}, bucket.name, 100)
		if err != nil {
			assert.Failf(t, "Failed to list S3 Buckets", errors.WithStackTrace(err).Error())
		} else {
			assert.NotContains(t, bucketNamesPerRegion[awsParams.region], aws.String(bucket.name))
			logging.Logger.Debugf("SUCCESS: Nuked bucket - %s", bucket.name)
		}
	}, nil
}

// TestNukeS3Bucket tests S3 bucket deletion
func TestNukeS3Bucket(t *testing.T) {
	t.Parallel()

	type testCaseStruct struct {
		name string
		args TestNukeS3BucketArgs
	}

	var allTestCases []testCaseStruct

	allTestCases = append(allTestCases, testCaseStruct{
		"EmptyBucket",
		TestNukeS3BucketArgs{isVersioned: false, checkDeleteMarker: false, objectCount: 0, objectBatchsize: 0, shouldNuke: true},
	})

	for _, bucketType := range []string{"NoVersioning", "Versioning"} {
		isVersioned := false
		if bucketType == "Versioning" {
			isVersioned = true
		}
		testCases := []testCaseStruct{
			{
				bucketType + "_AllObjects",
				TestNukeS3BucketArgs{
					isVersioned:       isVersioned,
					checkDeleteMarker: false,
					objectCount:       10,
					objectBatchsize:   1000,
					shouldNuke:        true,
				},
			},
			{
				bucketType + "_BatchObjects_ValidBatchSize",
				TestNukeS3BucketArgs{
					isVersioned:       isVersioned,
					checkDeleteMarker: false,
					objectCount:       10,
					objectBatchsize:   5,
					shouldNuke:        true,
				},
			},
			{
				bucketType + "_BatchObjects_InvalidBatchSize_Over",
				TestNukeS3BucketArgs{
					isVersioned:       isVersioned,
					checkDeleteMarker: false,
					objectCount:       2,
					objectBatchsize:   1001,
					shouldNuke:        false,
				},
			},
			{
				bucketType + "_BatchObjects_InvalidBatchSize_Under",
				TestNukeS3BucketArgs{
					isVersioned:       isVersioned,
					checkDeleteMarker: false,
					objectCount:       2,
					objectBatchsize:   0,
					shouldNuke:        false,
				},
			},
		}
		for _, tc := range testCases {
			allTestCases = append(allTestCases, tc)
		}
	}

	allTestCases = append(allTestCases, testCaseStruct{
		"Versioning_DeleteMarker",
		TestNukeS3BucketArgs{
			isVersioned:       true,
			checkDeleteMarker: true,
			objectCount:       10,
			objectBatchsize:   1000,
			shouldNuke:        true,
		},
	})

	for _, tc := range allTestCases {
		testFunc, err := genTestNukeS3Bucket(tc.args)
		if err != nil {
			assert.Fail(t, errors.WithStackTrace(err).Error())
		}
		t.Run(tc.name, testFunc)
	}
}
