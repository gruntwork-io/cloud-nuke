package aws

import (
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/accessanalyzer"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAccessAnalyzers(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	// We hard code the region here to avoid the tests colliding with each other, since we can only have one account
	// analyzer per region (but we can have multiple org analyzers).
	region := "us-west-1"

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := accessanalyzer.New(session)

	analyzerName := createAccessAnalyzer(t, svc)
	defer deleteAccessAnalyzer(t, svc, analyzerName, true)

	analyzerNames, err := getAllAccessAnalyzers(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(analyzerNames), aws.StringValue(analyzerName))
}

func TestTimeFilterExclusionNewlyCreatedAccessAnalyzer(t *testing.T) {
	t.Parallel()

	// We hard code the region here to avoid the tests colliding with each other, since we can only have one account
	// analyzer per region (but we can have multiple org analyzers).
	region := "us-west-2"

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := accessanalyzer.New(session)

	analyzerName := createAccessAnalyzer(t, svc)
	defer deleteAccessAnalyzer(t, svc, analyzerName, true)

	// Assert Access Analyzer is picked up without filters
	analyzerNamesNewer, err := getAllAccessAnalyzers(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(analyzerNamesNewer), aws.StringValue(analyzerName))

	// Assert analyzer doesn't appear when we look at users older than 1 Hour
	olderThan := time.Now().Add(-1 * time.Hour)
	analyzerNamesOlder, err := getAllAccessAnalyzers(session, olderThan, config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, aws.StringValueSlice(analyzerNamesOlder), aws.StringValue(analyzerName))
}

func TestNukeAccessAnalyzerOne(t *testing.T) {
	t.Parallel()

	// We hard code the region here to avoid the tests colliding with each other, since we can only have one account
	// analyzer per region (but we can have multiple org analyzers).
	region := "eu-west-1"

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := accessanalyzer.New(session)

	// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
	analyzerName := createAccessAnalyzer(t, svc)
	defer deleteAccessAnalyzer(t, svc, analyzerName, false)
	identifiers := []*string{analyzerName}

	require.NoError(
		t,
		nukeAllAccessAnalyzers(session, identifiers),
	)

	// Make sure the Access Analyzer is deleted.
	assertAccessAnalyzersDeleted(t, svc, identifiers)
}

// Helper functions for driving the Access Analyzer tests

// createAccessAnalyzer will create a new IAM Access Analyzer for test purposes
func createAccessAnalyzer(t *testing.T, svc *accessanalyzer.AccessAnalyzer) *string {
	name := fmt.Sprintf("cloud-nuke-test-%s", strings.ToLower(random.UniqueId()))
	resp, err := svc.CreateAnalyzer(&accessanalyzer.CreateAnalyzerInput{
		AnalyzerName: aws.String(name),
		Type:         aws.String("ACCOUNT"),
	})
	require.NoError(t, err)
	if resp.Arn == nil {
		t.Fatalf("Impossible error: AWS returned nil NAT gateway")
	}

	// AccessAnalyzer API operates on the name, so we need to extract out the name part of the ARN. Analyzer ARNs are of
	// the form: arn:aws:access-analyzer:sa-east-1:000000000000:analyzer/test-iam-access-analyzer-fhkb2x-sa_east_1, so
	// to get the name we can extract the resource part and split on `/` and return the second part.
	arn, err := arn.Parse(aws.StringValue(resp.Arn))
	require.NoError(t, err)
	nameParts := strings.Split(arn.Resource, "/")
	require.Equal(t, 2, len(nameParts))
	require.Equal(t, "analyzer", nameParts[0])
	return aws.String(nameParts[1])
}

// deleteAccessAnalyzer is a function to delete the given NAT gateway.
func deleteAccessAnalyzer(t *testing.T, svc *accessanalyzer.AccessAnalyzer, analyzerName *string, checkErr bool) {
	input := &accessanalyzer.DeleteAnalyzerInput{AnalyzerName: analyzerName}
	_, err := svc.DeleteAnalyzer(input)
	if checkErr {
		require.NoError(t, err)
	}
}

func assertAccessAnalyzersDeleted(t *testing.T, svc *accessanalyzer.AccessAnalyzer, identifiers []*string) {
	for _, identifier := range identifiers {
		_, err := svc.GetAnalyzer(&accessanalyzer.GetAnalyzerInput{AnalyzerName: identifier})
		if err == nil {
			t.Fatalf("Access Analyzer %s still exists", aws.StringValue(identifier))
		}
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == accessanalyzer.ErrCodeResourceNotFoundException {
			continue
		}
		t.Fatalf("Error checking for access analyzer %s: %s", aws.StringValue(identifier), err)
	}
}
