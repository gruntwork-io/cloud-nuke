package aws

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	err := SetEnvLogLevel()
	if err != nil {
		logging.Logger.Errorf("Invalid log level - %s", err)
		os.Exit(1)
	}
	exitVal := m.Run()
	os.Exit(exitVal)
}

// TestListS3Bucket represents arguments for TestListS3Bucket
type TestListS3BucketArgs struct {
	bucketTags  []map[string]string
	batchSize   int
	shouldError bool
	shouldMatch bool
}

// testListS3Bucket - helper function for TestListS3Bucket
func testListS3Bucket(t *testing.T, args TestListS3BucketArgs) {
	awsParams, err := NewCloudNukeAWSParams("")
	require.NoError(t, err, "Failed to setup AWS params")

	bucketName := S3GenBucketName()

	targetRegions := []string{*awsParams.AWSSession.Config.Region}

	// Please note that we are passing the same session that was used to create the bucket
	// This is required so that the defer cleanup call always gets the right bucket region
	// to delete
	defer nukeAllS3Buckets(awsParams.AWSSession, []*string{aws.String(bucketName)}, 1000)

	// Verify that - before creating bucket - it should not exist
	//
	// Please note that we are not reusing S3TestAWSParams.awsSession and creating a random session in a region other
	// than the one in which the bucket is created - this is useful to test the scenario where the user has
	// AWS_DEFAULT_REGION set to region x but the bucket is in region y.
	bucketNamesPerRegion, err := getAllS3Buckets(awsParams.AWSSession, time.Now().Add(1*time.Hour*-1), targetRegions, bucketName, args.batchSize, config.Config{})
	if args.shouldError {
		require.Error(t, err, "Did not fail for invalid batch size")
		logging.Logger.Debugf("SUCCESS: Did not list buckets due to invalid batch size - %s - %s", bucketName, err.Error())
		return
	}

	require.NoError(t, err, "Failed to list S3 Buckets")

	// Validate test bucket does not exist before creation
	require.NotContains(t, bucketNamesPerRegion[*awsParams.AWSSession.Config.Region], aws.String(bucketName))

	// Create test bucket
	var bucketTags []map[string]string
	if args.bucketTags != nil && len(args.bucketTags) > 0 {
		bucketTags = args.bucketTags
	}

	svc := s3.New(awsParams.AWSSession)
	err = S3CreateBucket(svc, bucketName, bucketTags, false)

	require.NoError(t, err, "Failed to create test buckets")

	bucketNamesPerRegion, err = getAllS3Buckets(awsParams.AWSSession, time.Now().Add(1*time.Hour), targetRegions, bucketName, args.batchSize, config.Config{})
	require.NoError(t, err, "Failed to list S3 Buckets")

	if args.shouldMatch {
		require.Contains(t, bucketNamesPerRegion[*awsParams.AWSSession.Config.Region], aws.String(bucketName))
		logging.Logger.Debugf("SUCCESS: Matched bucket - %s", bucketName)
	} else {
		require.NotContains(t, bucketNamesPerRegion[*awsParams.AWSSession.Config.Region], aws.String(bucketName))
		logging.Logger.Debugf("SUCCESS: Did not match bucket - %s", bucketName)
	}
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
		// Capture the range variable as per https://blog.golang.org/subtests
		// Not doing this will lead to tc being set to the last entry in the testCases
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testListS3Bucket(t, tc.args)
		})
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

// testNukeS3Bucket - generates the test function for TestNukeS3Bucket
func testNukeS3Bucket(t *testing.T, args TestNukeS3BucketArgs) {
	awsParams, err := NewCloudNukeAWSParams("")
	require.NoError(t, err, "Failed to setup AWS params")

	// Create test bucket
	bucketName := S3GenBucketName()
	var bucketTags []map[string]string

	svc := s3.New(awsParams.AWSSession)
	err = S3CreateBucket(svc, bucketName, bucketTags, args.isVersioned)
	require.NoError(t, err, "Failed to create test bucket")

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
				err := S3BucketAddObject(awsParams.AWSSession, bucketName, fileName, fileBody)
				require.NoError(t, err, "Failed to add object to test bucket")
			}
		}

		// Do a simple delete to create DeleteMarker object
		if args.checkDeleteMarker {
			targetObject := "l1/l2/l3/f0.txt"
			logging.Logger.Debugf("Bucket: %s - doing simple delete on object: %s", bucketName, targetObject)

			_, err = svc.DeleteObject(&s3.DeleteObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String("l1/l2/l3/f0.txt"),
			})
			require.NoError(t, err, "Failed to create delete marker")
		}
	}

	defer nukeAllS3Buckets(awsParams.AWSSession, []*string{aws.String(bucketName)}, 1000)

	// Nuke the test bucket
	delCount, err := nukeAllS3Buckets(awsParams.AWSSession, []*string{aws.String(bucketName)}, args.objectBatchsize)
	require.NoError(t, err, "Failed to nuke s3 buckets")

	// If we should not nuke the bucket then deleted bucket count should be 0
	if !args.shouldNuke {
		if delCount > 0 {
			require.Failf(t, "Should not nuke but got delCount > 0", "delCount: %d", delCount)
		}
		logging.Logger.Debugf("SUCCESS: Did not nuke bucket - %s", bucketName)
		return
	}

	var configObj *config.Config
	configObj, err = config.GetConfig("../config/mocks/s3_include_names.yaml")
	require.NoError(t, err)

	// Verify that - after nuking test bucket - it should not exist
	bucketNamesPerRegion, err := getAllS3Buckets(awsParams.AWSSession, time.Now().Add(1*time.Hour), []string{*awsParams.AWSSession.Config.Region}, bucketName, 100, *configObj)
	require.NoError(t, err, "Failed to list S3 Buckets")
	require.NotContains(t, bucketNamesPerRegion[*awsParams.AWSSession.Config.Region], aws.String(bucketName))
	logging.Logger.Debugf("SUCCESS: Nuked bucket - %s", bucketName)
}

// TestNukeS3Bucket tests S3 bucket deletion
func TestNukeS3Bucket(t *testing.T) {
	t.Parallel()

	type testCaseStruct struct {
		name string
		args TestNukeS3BucketArgs
	}

	var allTestCases []testCaseStruct

	for _, bucketType := range []string{"NoVersioning", "Versioning"} {
		isVersioned := bucketType == "Versioning"
		testCases := []testCaseStruct{
			{
				bucketType + "_EmptyBucket",
				TestNukeS3BucketArgs{
					isVersioned:       isVersioned,
					checkDeleteMarker: false,
					objectCount:       0,
					objectBatchsize:   0,
					shouldNuke:        true,
				},
			},
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
		// Capture the range variable as per https://blog.golang.org/subtests
		// Not doing this will lead to tc being set to the last entry in the testCases
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testNukeS3Bucket(t, tc.args)
		})
	}
}

// TestFilterS3BucketArgs represents arguments for TestFilterS3Bucket_Config
type TestFilterS3BucketArgs struct {
	configFilePath string
	matches        []string
}

func bucketNamesForConfigTests() []string {
	return []string{
		"alb-alb-123456-access-logs-" + S3GenBucketName(),
		"alb-alb-234567-access-logs-" + S3GenBucketName(),
		"tonico-prod-alb-access-logs-" + S3GenBucketName(),
		"prod-alb-public-access-logs-" + S3GenBucketName(),
		"stage-alb-internal-access-logs-" + S3GenBucketName(),
		"stage-alb-public-access-logs-" + S3GenBucketName(),
		"cloud-watch-logs-staging-" + S3GenBucketName(),
		"something-else-logs-staging-" + S3GenBucketName(),
	}
}

// TestFilterS3Bucket_Config tests listing only S3 buckets that match config file
func TestFilterS3Bucket_Config(t *testing.T) {
	t.Parallel()

	// Create AWS session in ca-central-1
	awsParams, err := NewCloudNukeAWSParams("ca-central-1")
	require.NoError(t, err, "Failed to setup AWS params")

	// Nuke all buckets in ca-central-1 first
	// passing in a config that matches all buckets
	var configObj *config.Config
	configObj, err = config.GetConfig("../config/mocks/s3_all.yaml")

	// Verify that only filtered buckets are listed
	cleanupBuckets, err := getAllS3Buckets(awsParams.AWSSession, time.Now().Add(1*time.Hour), []string{*awsParams.AWSSession.Config.Region}, "", 100, *configObj)
	require.NoError(t, err, "Failed to list S3 Buckets in ca-central-1")

	nukeAllS3Buckets(awsParams.AWSSession, cleanupBuckets[*awsParams.AWSSession.Config.Region], 1000)

	// Create test buckets in ca-central-1
	var bucketTags []map[string]string
	bucketNames := bucketNamesForConfigTests()
	svc := s3.New(awsParams.AWSSession)
	for _, bucketName := range bucketNames {
		err = S3CreateBucket(svc, bucketName, bucketTags, false)
		require.NoErrorf(t, err, "Failed to create test bucket - %s", bucketName)
	}

	// Please note that we are not reusing awsParams.awsSession and creating a random session in a region other
	// than the one in which the bucket is created - this is useful to test the scenario where the user has
	// AWS_DEFAULT_REGION set to region x but the bucket is in region y.
	awsParamsRand, err := NewCloudNukeAWSParams("")
	require.NoError(t, err, "Failed to create session in random region")

	// Define test cases
	type testCaseStruct struct {
		name string
		args TestFilterS3BucketArgs
	}

	includeBuckets := []string{}
	includeBuckets = append(includeBuckets, bucketNames[:4]...)

	excludeBuckets := []string{}
	excludeBuckets = append(excludeBuckets, bucketNames[:3]...)
	excludeBuckets = append(excludeBuckets, bucketNames[4])
	excludeBuckets = append(excludeBuckets, bucketNames[6:]...)

	filterBuckets := []string{}
	filterBuckets = append(filterBuckets, bucketNames[:3]...)

	testCases := []testCaseStruct{
		{
			"Include",
			TestFilterS3BucketArgs{
				configFilePath: "../config/mocks/s3_include_names.yaml",
				matches:        includeBuckets,
			},
		},
		{
			"Exclude",
			TestFilterS3BucketArgs{
				configFilePath: "../config/mocks/s3_exclude_names.yaml",
				matches:        excludeBuckets,
			},
		},
		{
			"IncludeAndExclude",
			TestFilterS3BucketArgs{
				configFilePath: "../config/mocks/s3_filter_names.yaml",
				matches:        filterBuckets,
			},
		},
	}

	// Clean up test buckets
	defer nukeAllS3Buckets(awsParams.AWSSession, aws.StringSlice(bucketNames), 1000)
	t.Run("config tests", func(t *testing.T) {
		for _, tc := range testCases {
			// Capture the range variable as per https://blog.golang.org/subtests
			// Not doing this will lead to tc being set to the last entry in the testCases
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				var configObj *config.Config
				configObj, err = config.GetConfig(tc.args.configFilePath)

				// Verify that only filtered buckets are listed (use random region)
				bucketNamesPerRegion, err := getAllS3Buckets(
					awsParamsRand.AWSSession,
					time.Now().Add(1*time.Hour), []string{*awsParams.AWSSession.Config.Region}, "", 100, *configObj,
				)

				require.NoError(t, err, "Failed to list S3 Buckets")
				require.Equal(t, len(tc.args.matches), len(bucketNamesPerRegion[*awsParamsRand.AWSSession.Config.Region]))
				require.Subset(t, aws.StringValueSlice(bucketNamesPerRegion[*awsParamsRand.AWSSession.Config.Region]), tc.args.matches)
			})
		}
	})
}
