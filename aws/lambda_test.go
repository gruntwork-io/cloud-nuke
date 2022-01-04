package aws

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: Move this into the utils package
func createZipFile(filename string, file string) error {
	err := ioutil.WriteFile(file, []byte("package test"), 0755)

	newZipFile, _ := os.Create(filename)
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	err = addFileToZip(zipWriter, file)
	if err != nil {
		return err
	}

	err = removeFile(file)
	if err != nil {
		return err
	}

	return nil
}

// TODO: Move this into the utils package
func removeFile(zipFileName string) error {
	err := os.Remove(zipFileName)
	if err != nil {
		return err
	}
	return nil
}

// TODO: Move this into the utils package
func addFileToZip(zipWriter *zip.Writer, filename string) error {
	fileToZip, err := os.Open(filename)
	defer fileToZip.Close()

	info, err := fileToZip.Stat()

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Name = filename
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	_, err = io.Copy(writer, fileToZip)
	return err
}

func createTestLambdaFunction(t *testing.T, session *session.Session, name string) {
	svc := lambda.New(session)

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	roleName := uniqueTestID + "-role"
	bucketName := uniqueTestID + "-bucket"
	zipFileName := uniqueTestID + ".zip"
	goFileName := uniqueTestID + ".go"

	// Prepare resources
	// Create the IAM roles for Lambda function
	role := createLambdaRole(t, session, roleName)
	defer deleteLambdaRole(session, role)

	// IAM resources are slow to propagate, so give it some
	// time
	time.Sleep(15 * time.Second)

	svcs3 := s3.New(session)
	svcs3.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})

	createZipFile(zipFileName, goFileName)
	defer removeFile(zipFileName)

	// Upload Zip
	reader := strings.NewReader(zipFileName)
	uploader := s3manager.NewUploader(session)
	uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(zipFileName),
		Body:   reader,
	})

	runTime := "go1.x"
	contents, err := ioutil.ReadFile(zipFileName)
	createCode := &lambda.FunctionCode{
		ZipFile: contents,
	}
	params := &lambda.CreateFunctionInput{
		Code:         createCode,
		FunctionName: &name,
		Handler:      awsgo.String(goFileName),
		Role:         role.Arn,
		Runtime:      &runTime,
	}

	_, err = svc.CreateFunction(params)
	require.NoError(t, err)
}

func TestLambdaFunctionConfigFile(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	includedLambdaFunctionName := "cloud-nuke-test-include-" + util.UniqueID()
	createTestLambdaFunction(t, session, includedLambdaFunctionName)

	excludedLambdaFunctionName := "cloud-nuke-test-" + util.UniqueID()
	createTestLambdaFunction(t, session, excludedLambdaFunctionName)

	defer nukeAllLambdaFunctions(session, []*string{&includedLambdaFunctionName, &excludedLambdaFunctionName})

	excludeAfter := time.Now().Add(1 * time.Hour)
	lambdaFunctions, err := getAllLambdaFunctions(session, excludeAfter, config.Config{
		LambdaFunction: config.ResourceType{
			IncludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{RE: *regexp.MustCompile("^cloud-nuke-test-include-.*")},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(lambdaFunctions))
	require.Equal(t, includedLambdaFunctionName, aws.StringValue(lambdaFunctions[0]))
}

func TestNukeLambdaFunction(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()

	require.NoError(t, errors.WithStackTrace(err))

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	lambdaFunctionName := "cloud-nuke-test-" + util.UniqueID()
	lambdaFunctionName2 := "cloud-nuke-test2-" + util.UniqueID()

	excludeAfter := time.Now().Add(1 * time.Hour)

	createTestLambdaFunction(t, session, lambdaFunctionName)
	createTestLambdaFunction(t, session, lambdaFunctionName2)

	defer func() {
		nukeAllLambdaFunctions(session, []*string{&lambdaFunctionName, &lambdaFunctionName2})

		lambdaFunctionNames, _ := getAllLambdaFunctions(session, excludeAfter, config.Config{}, 1)

		assert.NotContains(t, awsgo.StringValueSlice(lambdaFunctionNames), lambdaFunctionName)
		assert.NotContains(t, awsgo.StringValueSlice(lambdaFunctionNames), lambdaFunctionName2)

	}()

	lambdaFunctions, err := getAllLambdaFunctions(session, excludeAfter, config.Config{}, 1)

	if err != nil {
		assert.Failf(t, "Unable to fetch list of Lambda Functions", errors.WithStackTrace(err).Error())
	}

	assert.Contains(t, awsgo.StringValueSlice(lambdaFunctions), lambdaFunctionName)
	assert.Contains(t, awsgo.StringValueSlice(lambdaFunctions), lambdaFunctionName2)

}
