package aws

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// S3TestGenBucketName generates a test bucket name.
func S3TestGenBucketName() string {
	return strings.ToLower("cloud-nuke-test-" + util.UniqueID() + util.UniqueID())
}

// S3TestCreateNewAWSSession creates a new session for testing and returns it.
func S3TestCreateNewAWSSession(region string) (*session.Session, error) {
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

// S3TestAWSParams has AWS params info,
type S3TestAWSParams struct {
	region     string
	awsSession *session.Session
	svc        *s3.S3
}

// newS3TestAWSParams sets up common operations for nuke S3 tests.
func newS3TestAWSParams(region string) (S3TestAWSParams, error) {
	var params S3TestAWSParams

	if region == "" {
		var err error
		region, err = getRandomRegion()
		if err != nil {
			return params, err
		}
	}

	params.region = region

	awsSession, err := S3TestCreateNewAWSSession(params.region)
	if err != nil {
		return params, err
	}
	params.awsSession = awsSession

	params.svc = s3.New(params.awsSession)
	if err != nil {
		return params, err
	}

	return params, nil
}

// S3TestCreateBucket creates a test bucket and optionally tags and versions it.
func S3TestCreateBucket(svc *s3.S3, bucketName string, tags []map[string]string, isVersioned bool) error {
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

	err = svc.WaitUntilBucketExists(
		&s3.HeadBucketInput{
			Bucket: aws.String(bucketName),
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// S3TestBucketAddObject adds an object ot an S3 bucket.
func S3TestBucketAddObject(awsParams S3TestAWSParams, bucketName string, fileName string, fileBody string) error {
	logging.Logger.Debugf("Bucket: %s - adding object: %s - content: %s", bucketName, fileName, fileBody)

	reader := strings.NewReader(fileBody)
	uploader := s3manager.NewUploader(awsParams.awsSession)

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

// TestListS3Bucket represents arguments for TestListS3Bucket
type TestListS3BucketArgs struct {
	bucketTags  []map[string]string
	batchSize   int
	shouldError bool
	shouldMatch bool
}

// testListS3Bucket - helper function for TestListS3Bucket
func testListS3Bucket(t *testing.T, args TestListS3BucketArgs) {
	awsParams, err := newS3TestAWSParams("")
	require.NoError(t, err, "Failed to setup AWS params")

	bucketName := S3TestGenBucketName()

	awsSession, err := S3TestCreateNewAWSSession("")
	require.NoError(t, err, "Failed to create random session")

	targetRegions := []string{awsParams.region}

	// Please note that we are passing the same session that was used to create the bucket
	// This is required so that the defer cleanup call always gets the right bucket region
	// to delete
	defer func() {
		_, err := nukeAllS3Buckets(awsParams.awsSession, []*string{aws.String(bucketName)}, 1000)
		if args.shouldError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}()

	// Verify that - before creating bucket - it should not exist
	//
	// Please note that we are not reusing S3TestAWSParams.awsSession and creating a random session in a region other
	// than the one in which the bucket is created - this is useful to test the scenario where the user has
	// AWS_DEFAULT_REGION set to region x but the bucket is in region y.
	bucketNamesPerRegion, err := getAllS3Buckets(awsSession, time.Now().Add(1*time.Hour*-1), targetRegions, bucketName, args.batchSize, config.Config{})
	if args.shouldError {
		require.Error(t, err)
		logging.Logger.Debugf("SUCCESS: Did not list buckets due to invalid batch size - %s - %s", bucketName, err.Error())
		return
	}

	require.NoError(t, err, "Failed to list S3 Buckets")

	// Validate test bucket does not exist before creation
	require.NotContains(t, bucketNamesPerRegion[awsParams.region], aws.String(bucketName))

	// Create test bucket
	var bucketTags []map[string]string
	if args.bucketTags != nil && len(args.bucketTags) > 0 {
		bucketTags = args.bucketTags
	}

	err = S3TestCreateBucket(awsParams.svc, bucketName, bucketTags, false)

	require.NoError(t, err, "Failed to create test buckets")

	bucketNamesPerRegion, err = getAllS3Buckets(awsSession, time.Now().Add(1*time.Hour), targetRegions, bucketName, args.batchSize, config.Config{})
	require.NoError(t, err, "Failed to list S3 Buckets")

	if args.shouldMatch {
		require.Contains(t, bucketNamesPerRegion[awsParams.region], aws.String(bucketName))
		logging.Logger.Debugf("SUCCESS: Matched bucket - %s", bucketName)
	} else {
		require.NotContains(t, bucketNamesPerRegion[awsParams.region], aws.String(bucketName))
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
	shouldError       bool
}

// testNukeS3Bucket - generates the test function for TestNukeS3Bucket
func testNukeS3Bucket(t *testing.T, args TestNukeS3BucketArgs) {
	awsParams, err := newS3TestAWSParams("eu-central-1")
	require.NoError(t, err, "Failed to setup AWS params")

	// Create test bucket
	bucketName := S3TestGenBucketName()
	var bucketTags []map[string]string

	err = S3TestCreateBucket(awsParams.svc, bucketName, bucketTags, args.isVersioned)
	require.NoError(t, err, "Failed to create test bucket")

	awsSession, err := S3TestCreateNewAWSSession("")
	require.NoError(t, err, "Failed to create random session")

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
				err := S3TestBucketAddObject(awsParams, bucketName, fileName, fileBody)
				require.NoError(t, err, "Failed to add object to test bucket")
			}
		}

		// Do a simple delete to create DeleteMarker object
		if args.checkDeleteMarker {
			targetObject := "l1/l2/l3/f0.txt"
			logging.Logger.Debugf("Bucket: %s - doing simple delete on object: %s", bucketName, targetObject)

			_, err = awsParams.svc.DeleteObject(&s3.DeleteObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String("l1/l2/l3/f0.txt"),
			})
			require.NoError(t, err, "Failed to create delete marker")
		}
	}

	// Don't remove this.
	// It ensures that all S3 buckets created as part of this test will be nuked after the test has run.
	// This is necessary, as some test cases are expected to fail & test that the buckets with invalid args are not nuked.
	// For more details, look at Github issue-140: https://github.com/gruntwork-io/cloud-nuke/issues/140
	defer nukeAllS3Buckets(awsParams.awsSession, []*string{aws.String(bucketName)}, 1000)

	// Nuke the test bucket
	delCount, err := nukeAllS3Buckets(awsParams.awsSession, []*string{aws.String(bucketName)}, args.objectBatchsize)
	if args.shouldError {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
	}

	// If we should not nuke the bucket then deleted bucket count should be 0
	if !args.shouldNuke {
		if delCount > 0 {
			require.Failf(t, "Should not nuke but got delCount > 0", "delCount: %d", delCount)
		}
		logging.Logger.Debugf("SUCCESS: Did not nuke bucket - %s", bucketName)
		return
	}

	var configObj *config.Config
	configObj, err = config.GetConfig(readTemplate(t, "../config/mocks/s3_include_names.yaml", map[string]string{"__TESTID__": ""}))
	require.NoError(t, err)

	// Verify that - after nuking test bucket - it should not exist
	bucketNamesPerRegion, err := getAllS3Buckets(awsSession, time.Now().Add(1*time.Hour), []string{awsParams.region}, bucketName, 100, *configObj)
	require.NoError(t, err, "Failed to list S3 Buckets")
	require.NotContains(t, bucketNamesPerRegion[awsParams.region], aws.String(bucketName))
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
					objectBatchsize:   1,
					shouldNuke:        true,
					shouldError:       false,
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
					shouldError:       false,
				},
			},
			{
				bucketType + "_BatchObjects_ValidBatchSize",
				TestNukeS3BucketArgs{
					isVersioned:       isVersioned,
					checkDeleteMarker: false,
					objectCount:       30,
					objectBatchsize:   5,
					shouldNuke:        true,
					shouldError:       false,
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
					shouldError:       true,
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
					shouldError:       true,
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

// readTemplate - read and replace variables in template, return path to temporary file with processed template.
func readTemplate(t *testing.T, templatePath string, variables map[string]string) string {
	_, name := path.Split(templatePath)
	file, err := ioutil.TempFile(os.TempDir(), "*"+name)
	require.NoError(t, err)

	defer file.Close()
	content, err := ioutil.ReadFile(templatePath)
	require.NoError(t, err)

	template := string(content)

	for key, value := range variables {
		template = strings.Replace(template, key, value, -1)
	}

	_, err = file.WriteString(template)
	require.NoError(t, err)

	return file.Name()
}

// TestFilterS3BucketArgs represents arguments for TestFilterS3Bucket_Config
type TestFilterS3BucketArgs struct {
	configFilePath string
	exactMatch     bool
	matches        []string
}

func bucketNamesForConfigTests(id string) []string {
	return []string{
		"alb-alb-123456-access-logs-" + id,
		"alb-alb-234567-access-logs-" + id,
		"tonico-prod-alb-access-logs-" + id,
		"prod-alb-public-access-logs-" + id,
		"stage-alb-internal-access-logs-" + id,
		"stage-alb-public-access-logs-" + id,
		"cloud-watch-logs-staging-" + id,
		"something-else-logs-staging-" + id,
	}
}

// TestFilterS3Bucket_Config tests listing only S3 buckets that match config file
func TestFilterS3Bucket_Config(t *testing.T) {
	t.Parallel()

	testId := S3TestGenBucketName()
	logging.Logger.Debugf("Generated test id %v", testId)

	// Create AWS session in ca-central-1
	awsParams, err := newS3TestAWSParams("ca-central-1")
	require.NoError(t, err, "Failed to setup AWS params")

	// Nuke all buckets in ca-central-1 first
	// passing in a config that matches all buckets
	var configObj *config.Config
	configObj, err = config.GetConfig(readTemplate(t, "../config/mocks/s3_all.yaml", map[string]string{"__TESTID__": testId}))

	// Verify that only filtered buckets are listed
	cleanupBuckets, err := getAllS3Buckets(awsParams.awsSession, time.Now().Add(1*time.Hour), []string{awsParams.region}, "", 100, *configObj)
	require.NoError(t, err, "Failed to list S3 Buckets in ca-central-1")

	_, err = nukeAllS3Buckets(awsParams.awsSession, cleanupBuckets[awsParams.region], 1000)
	require.NoError(t, err)

	// Create test buckets in ca-central-1
	var bucketTags []map[string]string
	bucketNames := bucketNamesForConfigTests(testId)
	for _, bucketName := range bucketNames {
		err = S3TestCreateBucket(awsParams.svc, bucketName, bucketTags, false)
		require.NoErrorf(t, err, "Failed to create test bucket - %s", bucketName)
	}

	// Please note that we are not reusing awsParams.awsSession and creating a random session in a region other
	// than the one in which the bucket is created - this is useful to test the scenario where the user has
	// AWS_DEFAULT_REGION set to region x but the bucket is in region y.
	awsSession, err := S3TestCreateNewAWSSession("")
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
				configFilePath: readTemplate(t, "../config/mocks/s3_include_names.yaml", map[string]string{"__TESTID__": testId}),
				matches:        includeBuckets,
			},
		},
		{
			"Exclude",
			TestFilterS3BucketArgs{
				configFilePath: readTemplate(t, "../config/mocks/s3_exclude_names.yaml", map[string]string{"__TESTID__": testId}),
				matches:        excludeBuckets,
				// exclude match may include multiple buckets than created during test
				// https://github.com/gruntwork-io/cloud-nuke/issues/142
				exactMatch: false,
			},
		},
		{
			"IncludeAndExclude",
			TestFilterS3BucketArgs{
				configFilePath: readTemplate(t, "../config/mocks/s3_filter_names.yaml", map[string]string{"__TESTID__": testId}),
				matches:        filterBuckets,
			},
		},
	}

	// Clean up test buckets
	defer func() {
		_, err := nukeAllS3Buckets(awsParams.awsSession, aws.StringSlice(bucketNames), 1000)
		assert.NoError(t, err)
	}()
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
				bucketNamesPerRegion, err := getAllS3Buckets(awsSession, time.Now().Add(1*time.Hour), []string{awsParams.region}, "", 100, *configObj)

				require.NoError(t, err, "Failed to list S3 Buckets")
				if tc.args.exactMatch {
					require.Equal(t, len(tc.args.matches), len(bucketNamesPerRegion[awsParams.region]))
				} else {
					// in case of not exact match, at least check if number of matched buckets are more or equal to arg count
					require.True(t, len(bucketNamesPerRegion[awsParams.region]) >= len(tc.args.matches))
				}
				require.Subset(t, aws.StringValueSlice(bucketNamesPerRegion[awsParams.region]), tc.args.matches)
			})
		}
	})
}

// TestNukeS3BucketWithBucketPolicy tests deletion of S3 buckets with a policy that denies deletion
func TestNukeS3BucketWithBucketPolicy(t *testing.T) {
	awsParams, err := newS3TestAWSParams("")
	require.NoError(t, err, "Failed to setup AWS params")

	// Create test bucket
	bucketName := S3TestGenBucketName()
	var bucketTags []map[string]string

	err = S3TestCreateBucket(awsParams.svc, bucketName, bucketTags, false)
	require.NoError(t, err, "Failed to create test bucket")

	policy, _ := json.Marshal(map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect":    "Deny",
				"Principal": "*",
				"Action": []string{
					"s3:DeleteBucket",
				},
				"Resource": []string{
					fmt.Sprintf("arn:aws:s3:::%s", bucketName),
				},
			},
		},
	})

	_, err = awsParams.svc.PutBucketPolicy(&s3.PutBucketPolicyInput{
		Bucket: aws.String(bucketName),
		Policy: aws.String(string(policy)),
	})
	require.NoError(t, err)

	defer func() {
		/*
			If the policy was not removed, delete it manually and delete
			the bucket to not leave any test data in the account
		*/
		awsParams.svc.DeleteBucketPolicy(&s3.DeleteBucketPolicyInput{
			Bucket: aws.String(bucketName),
		})
		nukeAllS3Buckets(awsParams.awsSession, []*string{aws.String(bucketName)}, 1000)
	}()

	_, err = nukeAllS3Buckets(awsParams.awsSession, []*string{aws.String(bucketName)}, 1000)
	require.NoError(t, err)

}
