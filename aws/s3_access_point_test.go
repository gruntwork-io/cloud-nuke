package aws

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3control"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNukeS3AccessPointOne(t *testing.T) {
	t.Parallel()

	telemetry.InitTelemetry("cloud-nuke", "", "")

	region, err := getRandomRegion()
	require.NoError(t, err, "Failed to get random region")

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err, "Failed to create AWS session")

	accountId, err := util.GetCurrentAccountId(session)
	require.NoError(t, err, "Failed to get account id")

	s3Svc := s3.New(session)
	s3ControlSvc := s3control.New(session)

	// Create a new bucket for testing
	bucketName := S3TestGenBucketName()
	var bucketTags []map[string]string

	err = S3TestCreateBucket(s3Svc, bucketName, bucketTags, false)
	require.NoError(t, err, "Failed to create test bucket")
	defer func() {
		_, err := nukeAllS3Buckets(session, []*string{aws.String(bucketName)}, 1000)
		assert.NoError(t, err)
	}()

	// Create a new access point for testing
	s3AccessPoint := createS3AccessPoint(t, s3ControlSvc, accountId, bucketName)
	defer deleteS3AccessPoint(t, s3ControlSvc, s3AccessPoint, accountId, false)
	identifiers := []*string{s3AccessPoint}

	require.NoError(
		t,
		nukeAllS3AccessPoints(session, identifiers),
	)

	// Make sure the S3 Access Point is deleted.
	assertS3AccessPointsDeleted(t, s3ControlSvc, identifiers, accountId)
}

func TestNukeS3AccessPointsMoreThanOne(t *testing.T) {
	t.Parallel()

	telemetry.InitTelemetry("cloud-nuke", "", "")

	region, err := getRandomRegion()
	require.NoError(t, err, "Failed to get random region")

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err, "Failed to create AWS session")

	accountId, err := util.GetCurrentAccountId(session)
	require.NoError(t, err, "Failed to get account id")

	s3Svc := s3.New(session)
	s3ControlSvc := s3control.New(session)

	// Create a new bucket for testing
	bucketName := S3TestGenBucketName()
	var bucketTags []map[string]string

	err = S3TestCreateBucket(s3Svc, bucketName, bucketTags, false)
	require.NoError(t, err, "Failed to create test bucket")
	defer func() {
		_, err := nukeAllS3Buckets(session, []*string{aws.String(bucketName)}, 1000)
		assert.NoError(t, err)
	}()

	// Create new access points for testing
	s3AccessPoints := []*string{}
	for i := 0; i < 3; i++ {
		s3AccessPoint := createS3AccessPoint(t, s3ControlSvc, accountId, bucketName)
		defer deleteS3AccessPoint(t, s3ControlSvc, s3AccessPoint, accountId, false)
		s3AccessPoints = append(s3AccessPoints, s3AccessPoint)
	}

	require.NoError(
		t,
		nukeAllS3AccessPoints(session, s3AccessPoints),
	)

	// Make sure the S3 Access Point is deleted.
	assertS3AccessPointsDeleted(t, s3ControlSvc, s3AccessPoints, accountId)
}

func TestNukeS3ObjectLambdaAccessPointOne(t *testing.T) {
	t.Parallel()

	telemetry.InitTelemetry("cloud-nuke", "", "")

	region, err := getRandomRegion()
	require.NoError(t, err, "Failed to get random region")

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err, "Failed to create AWS session")

	accountId, err := util.GetCurrentAccountId(session)
	require.NoError(t, err, "Failed to get account id")

	s3Svc := s3.New(session)
	s3ControlSvc := s3control.New(session)

	// Create a new bucket for testing
	bucketName := S3TestGenBucketName()
	var bucketTags []map[string]string

	err = S3TestCreateBucket(s3Svc, bucketName, bucketTags, false)
	require.NoError(t, err, "Failed to create test bucket")
	defer func() {
		_, err := nukeAllS3Buckets(session, []*string{aws.String(bucketName)}, 1000)
		assert.NoError(t, err)
	}()

	// Create a new access point for testing
	s3AccessPoint := createS3AccessPoint(t, s3ControlSvc, accountId, bucketName)
	s3AccessPointArn := getS3AccessPointArn(t, session, s3AccessPoint, accountId)
	defer deleteS3AccessPoint(t, s3ControlSvc, s3AccessPoint, accountId, false)
	defer func() {
		err := nukeAllS3AccessPoints(session, []*string{s3AccessPoint})
		assert.NoError(t, err)
	}()

	// Create a new lambda function for testing
	functionName := "cloud-nuke-test-" + util.UniqueID()
	createTestLambdaFunction(t, session, functionName)
	functionArn := getLambdaFunctionArn(t, session, functionName)
	defer func() {
		err := nukeAllLambdaFunctions(session, aws.StringSlice([]string{functionName}))
		assert.NoError(t, err)
	}()

	// Create a new object lambda access point for testing
	s3ObjectLambdaAccessPoint := createS3ObjectLambdaAccessPoint(t, s3ControlSvc, accountId, s3AccessPointArn, functionArn)
	defer deleteS3ObjectLambdaAccessPoint(t, s3ControlSvc, s3ObjectLambdaAccessPoint, accountId, false)
	identifiers := []*string{s3ObjectLambdaAccessPoint}

	require.NoError(
		t,
		nukeAllS3ObjectLambdaAccessPoints(session, identifiers),
	)

	// Make sure the S3 Access Point is deleted.
	assertS3ObjectLambdaAccessPointDeleted(t, s3ControlSvc, identifiers, accountId)
}

func TestNukeS3ObjectLambdaAccessPointsMoreThanOne(t *testing.T) {
	t.Parallel()

	telemetry.InitTelemetry("cloud-nuke", "", "")

	region, err := getRandomRegion()
	require.NoError(t, err, "Failed to get random region")

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err, "Failed to create AWS session")

	accountId, err := util.GetCurrentAccountId(session)
	require.NoError(t, err, "Failed to get account id")

	s3Svc := s3.New(session)
	s3ControlSvc := s3control.New(session)

	// Create a new bucket for testing
	bucketName := S3TestGenBucketName()
	var bucketTags []map[string]string

	err = S3TestCreateBucket(s3Svc, bucketName, bucketTags, false)
	require.NoError(t, err, "Failed to create test bucket")
	defer func() {
		_, err := nukeAllS3Buckets(session, []*string{aws.String(bucketName)}, 1000)
		assert.NoError(t, err)
	}()

	// Create a new access point for testing
	s3AccessPoint := createS3AccessPoint(t, s3ControlSvc, accountId, bucketName)
	s3AccessPointArn := getS3AccessPointArn(t, session, s3AccessPoint, accountId)
	defer deleteS3AccessPoint(t, s3ControlSvc, s3AccessPoint, accountId, false)
	defer func() {
		err := nukeAllS3AccessPoints(session, []*string{s3AccessPoint})
		assert.NoError(t, err)
	}()

	// Create a new lambda function for testing
	functionName := "cloud-nuke-test-" + util.UniqueID()
	createTestLambdaFunction(t, session, functionName)
	functionArn := getLambdaFunctionArn(t, session, functionName)
	defer func() {
		err := nukeAllLambdaFunctions(session, aws.StringSlice([]string{functionName}))
		assert.NoError(t, err)
	}()

	// Create new object lambda access points for testing
	s3ObjectLambdaAccessPoints := []*string{}
	for i := 0; i < 3; i++ {
		s3ObjectLambdaAccessPoint := createS3ObjectLambdaAccessPoint(t, s3ControlSvc, accountId, s3AccessPointArn, functionArn)
		defer deleteS3ObjectLambdaAccessPoint(t, s3ControlSvc, s3ObjectLambdaAccessPoint, accountId, false)
		s3ObjectLambdaAccessPoints = append(s3ObjectLambdaAccessPoints, s3ObjectLambdaAccessPoint)
	}

	require.NoError(
		t,
		nukeAllS3ObjectLambdaAccessPoints(session, s3ObjectLambdaAccessPoints),
	)

	// Make sure the S3 Access Point is deleted.
	assertS3ObjectLambdaAccessPointDeleted(t, s3ControlSvc, s3ObjectLambdaAccessPoints, accountId)
}

func TestNukeS3MultiRegionAccessPointOne(t *testing.T) {
	t.Parallel()

	telemetry.InitTelemetry("cloud-nuke", "", "")

	region, err := getRandomRegion()
	require.NoError(t, err, "Failed to get random region")

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err, "Failed to create AWS session for s3 bucket")

	accountId, err := util.GetCurrentAccountId(session)
	require.NoError(t, err, "Failed to get account id")

	s3Svc := s3.New(session)
	// NOTE: this actions (about multi region access point) will always be routed to the US West (Oregon) Region.
	s3ControlSvc := s3control.New(session, &aws.Config{Region: aws.String("us-west-2")})

	// Create a new bucket for testing
	bucketName := S3TestGenBucketName()
	var bucketTags []map[string]string

	err = S3TestCreateBucket(s3Svc, bucketName, bucketTags, false)
	require.NoError(t, err, "Failed to create test bucket")
	defer func() {
		_, err := nukeAllS3Buckets(session, []*string{aws.String(bucketName)}, 1000)
		assert.NoError(t, err)
	}()

	// Create a new multi region access point for testing
	s3MultiRegionAccessPoint := createS3MultiRegionAccessPoint(t, s3ControlSvc, accountId, bucketName)
	defer deleteS3AccessPoint(t, s3ControlSvc, s3MultiRegionAccessPoint, accountId, false)
	identifiers := []*string{s3MultiRegionAccessPoint}

	require.NoError(
		t,
		nukeAllS3MultiRegionAccessPoints(session, identifiers),
	)

	// Make sure the S3 Access Point is deleted.
	assertS3MultiRegionAccessPointsDeleted(t, s3ControlSvc, identifiers, accountId)
}

func TestNukeS3MultiRegionAccessPointsMoreThanOne(t *testing.T) {
	t.Parallel()

	telemetry.InitTelemetry("cloud-nuke", "", "")

	region, err := getRandomRegion()
	require.NoError(t, err, "Failed to get random region")

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err, "Failed to create AWS session for s3 bucket")

	accountId, err := util.GetCurrentAccountId(session)
	require.NoError(t, err, "Failed to get account id")

	s3Svc := s3.New(session)
	// NOTE: this actions (about multi region access point) will always be routed to the US West (Oregon) Region.
	s3ControlSvc := s3control.New(session, &aws.Config{Region: aws.String("us-west-2")})

	// Create a new bucket for testing
	bucketName := S3TestGenBucketName()
	var bucketTags []map[string]string

	err = S3TestCreateBucket(s3Svc, bucketName, bucketTags, false)
	require.NoError(t, err, "Failed to create test bucket")
	defer func() {
		_, err := nukeAllS3Buckets(session, []*string{aws.String(bucketName)}, 1000)
		assert.NoError(t, err)
	}()

	// Create new multi region access points for testing
	s3MultiRegionAccessPoints := []*string{}
	for i := 0; i < 3; i++ {
		s3MultiRegionAccessPoint := createS3MultiRegionAccessPoint(t, s3ControlSvc, accountId, bucketName)
		defer deleteS3AccessPoint(t, s3ControlSvc, s3MultiRegionAccessPoint, accountId, false)
		s3MultiRegionAccessPoints = append(s3MultiRegionAccessPoints, s3MultiRegionAccessPoint)
	}

	require.NoError(
		t,
		nukeAllS3MultiRegionAccessPoints(session, s3MultiRegionAccessPoints),
	)

	// Make sure the S3 Access Points is deleted.
	assertS3MultiRegionAccessPointsDeleted(t, s3ControlSvc, s3MultiRegionAccessPoints, accountId)
}

// createS3AccessPoint will create a new S3 Access Point.
func createS3AccessPoint(t *testing.T, svc *s3control.S3Control, accountId string, bucketName string) *string {
	uniqueID := strings.ToLower(util.UniqueID()) // Access point name must not contain uppercase characters
	name := fmt.Sprintf("cloud-nuke-testing-%s", uniqueID)

	_, err := svc.CreateAccessPoint(&s3control.CreateAccessPointInput{
		AccountId: aws.String(accountId),
		Name:      aws.String(name),
		Bucket:    aws.String(bucketName),
		PublicAccessBlockConfiguration: &s3control.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(true),
			BlockPublicPolicy:     aws.Bool(true),
			IgnorePublicAcls:      aws.Bool(true),
			RestrictPublicBuckets: aws.Bool(true),
		},
	})
	require.NoError(t, err, "Failed to create test s3 access point")

	// Verify that the access point is generated well
	resp, err := svc.GetAccessPoint(&s3control.GetAccessPointInput{
		AccountId: aws.String(accountId),
		Name:      aws.String(name),
	})
	require.NoError(t, err, "Failed to get test s3 access point")
	if aws.StringValue(resp.Name) != name {
		t.Fatalf("Error creating S3 Access Point %s", name)
	}

	// Add an arbitrary sleep to account for eventual consistency
	time.Sleep(15 * time.Second)
	return &name
}

// createS3ObjectLambdaAccessPoint will create a new S3 Object Lambda Access Point.
func createS3ObjectLambdaAccessPoint(t *testing.T, svc *s3control.S3Control, accountId string, accessPointArn *string, functionArn *string) *string {
	uniqueID := strings.ToLower(util.UniqueID()) // Lambda Object Access point name must not contain uppercase characters
	name := fmt.Sprintf("cloud-nuke-testing-%s", uniqueID)

	_, err := svc.CreateAccessPointForObjectLambda(&s3control.CreateAccessPointForObjectLambdaInput{
		AccountId: aws.String(accountId),
		Name:      aws.String(name),
		Configuration: &s3control.ObjectLambdaConfiguration{
			AllowedFeatures: aws.StringSlice([]string{
				s3control.ObjectLambdaAllowedFeatureGetObjectRange,
				s3control.ObjectLambdaAllowedFeatureGetObjectPartNumber,
				s3control.ObjectLambdaAllowedFeatureHeadObjectRange,
				s3control.ObjectLambdaAllowedFeatureHeadObjectPartNumber,
			}),
			CloudWatchMetricsEnabled: aws.Bool(false),
			SupportingAccessPoint:    accessPointArn,
			TransformationConfigurations: []*s3control.ObjectLambdaTransformationConfiguration{
				{
					Actions: aws.StringSlice([]string{
						s3control.ObjectLambdaTransformationConfigurationActionGetObject,
						s3control.ObjectLambdaTransformationConfigurationActionHeadObject,
						s3control.ObjectLambdaTransformationConfigurationActionListObjects,
						s3control.ObjectLambdaTransformationConfigurationActionListObjectsV2,
					}),
					ContentTransformation: &s3control.ObjectLambdaContentTransformation{
						AwsLambda: &s3control.AwsLambdaTransformation{
							FunctionArn: functionArn,
						},
					},
				},
			},
		},
	})
	require.NoError(t, err, "Failed to create test s3 object lambda access point")

	// Verify that the object lambda access point is generated well
	resp, err := svc.GetAccessPointForObjectLambda(&s3control.GetAccessPointForObjectLambdaInput{
		AccountId: aws.String(accountId),
		Name:      aws.String(name),
	})
	require.NoError(t, err, "Failed to get test s3 object lambda access point")
	if aws.StringValue(resp.Name) != name {
		t.Fatalf("Error creating S3 Object Lambda Access Point %s", name)
	}

	// Add an arbitrary sleep to account for eventual consistency
	time.Sleep(15 * time.Second)
	return &name
}

// createS3MultiRegionAccessPoint will create a new S3 Multi Region Access Point.
func createS3MultiRegionAccessPoint(t *testing.T, svc *s3control.S3Control, accountId string, bucketName string) *string {
	uniqueID := strings.ToLower(util.UniqueID()) // Multi Region Access point name must not contain uppercase characters
	name := fmt.Sprintf("cloud-nuke-testing-%s", uniqueID)

	_, err := svc.CreateMultiRegionAccessPoint(&s3control.CreateMultiRegionAccessPointInput{
		AccountId: aws.String(accountId),
		Details: &s3control.CreateMultiRegionAccessPointInput_{
			Name: aws.String(name),
			PublicAccessBlock: &s3control.PublicAccessBlockConfiguration{
				BlockPublicAcls:       aws.Bool(true),
				BlockPublicPolicy:     aws.Bool(true),
				IgnorePublicAcls:      aws.Bool(true),
				RestrictPublicBuckets: aws.Bool(true),
			},
			Regions: []*s3control.Region{
				{
					Bucket: aws.String(bucketName),
				},
			},
		},
	})
	require.NoError(t, err, "Failed to create test s3 multi region access point")

	time.Sleep(10 * time.Second) // wait 10 seconds to get multi region access point

	err = waitUntilS3MultiRegionAccessPoint(svc, accountId, name)
	require.NoError(t, err, "Failed to wait s3 multi region access point creation")

	// Verify that the multi region access point is generated well
	resp, err := svc.GetMultiRegionAccessPoint(&s3control.GetMultiRegionAccessPointInput{
		AccountId: aws.String(accountId),
		Name:      aws.String(name),
	})
	require.NoError(t, err, "Failed to get test s3 multi region access point")
	if aws.StringValue(resp.AccessPoint.Name) != name {
		t.Fatalf("Error creating S3 Multi Region Access Point %s", name)
	}

	// Add an arbitrary sleep to account for eventual consistency
	time.Sleep(15 * time.Second)
	return &name
}

// deleteS3AccessPoint is a function to delete the given S3 Access Point.
func deleteS3AccessPoint(t *testing.T, svc *s3control.S3Control, name *string, accountId string, checkErr bool) {
	input := &s3control.DeleteAccessPointInput{
		AccountId: aws.String(accountId),
		Name:      name,
	}
	_, err := svc.DeleteAccessPoint(input)
	if checkErr {
		require.NoError(t, err)
	}
}

// deleteS3ObjectLambdaAccessPoint is a function to delete the given S3 Object Lambda Access Point.
func deleteS3ObjectLambdaAccessPoint(t *testing.T, svc *s3control.S3Control, name *string, accountId string, checkErr bool) {
	input := &s3control.DeleteAccessPointForObjectLambdaInput{
		AccountId: aws.String(accountId),
		Name:      name,
	}
	_, err := svc.DeleteAccessPointForObjectLambda(input)
	if checkErr {
		require.NoError(t, err)
	}
}

func assertS3AccessPointsDeleted(t *testing.T, svc *s3control.S3Control, identifiers []*string, accountId string) {
	for _, name := range identifiers {
		_, err := svc.GetAccessPoint(&s3control.GetAccessPointInput{
			AccountId: aws.String(accountId),
			Name:      name,
		})
		require.ErrorContainsf(t, err, "NoSuchAccessPoint", err.Error())
	}
}

func assertS3ObjectLambdaAccessPointDeleted(t *testing.T, svc *s3control.S3Control, identifiers []*string, accountId string) {
	for _, name := range identifiers {
		_, err := svc.GetAccessPointForObjectLambda(&s3control.GetAccessPointForObjectLambdaInput{
			AccountId: aws.String(accountId),
			Name:      name,
		})
		require.ErrorContainsf(t, err, "NoSuchAccessPoint", err.Error())
	}
}

func assertS3MultiRegionAccessPointsDeleted(t *testing.T, svc *s3control.S3Control, identifiers []*string, accountId string) {
	for _, name := range identifiers {
		_, err := svc.GetMultiRegionAccessPoint(&s3control.GetMultiRegionAccessPointInput{
			AccountId: aws.String(accountId),
			Name:      name,
		})
		require.ErrorContainsf(t, err, "NoSuchMultiRegionAccessPoint", err.Error())
	}
}

// getS3AccessPointArn is a function to get the access point's arn through access point name.
// To create s3 object lambda access point for testing, it should need an arn, not a name.
func getS3AccessPointArn(t *testing.T, session *session.Session, name *string, accountId string) *string {
	svc := s3control.New(session)

	resp, err := svc.GetAccessPoint(&s3control.GetAccessPointInput{
		AccountId: aws.String(accountId),
		Name:      name,
	})
	require.NoError(t, err, "Failed to get the s3 access point's arn")

	return resp.AccessPointArn
}

// getLambdaFunctionArn is a function to get the lambda function's arn through function name.
// To create s3 object lambda access point for testing, it should need an arn, not a name.
func getLambdaFunctionArn(t *testing.T, session *session.Session, name string) *string {
	svc := lambda.New(session)

	resp, err := svc.GetFunction(&lambda.GetFunctionInput{
		FunctionName: aws.String(name),
	})
	require.NoError(t, err, "Failed to get the lambda function's arn")

	return resp.Configuration.FunctionArn
}

// waitUntilS3MultiRegionAccessPoint is a function to wait multi region access point to create.
func waitUntilS3MultiRegionAccessPoint(svc *s3control.S3Control, accountId string, name string) error {
	input := &s3control.GetMultiRegionAccessPointInput{
		AccountId: aws.String(accountId),
		Name:      aws.String(name),
	}

	for i := 0; i < 240; i++ {
		multiRegionAccessPoint, err := svc.GetMultiRegionAccessPoint(input)
		status := multiRegionAccessPoint.AccessPoint.Status

		if aws.StringValue(status) != "CREATING" {
			return nil
		}

		if err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
		logging.Logger.Debug("Waiting for S3 Multi Region Access Point to be created")
	}

	return S3MultiRegionAccessPointDeleteError{name: name}
}
