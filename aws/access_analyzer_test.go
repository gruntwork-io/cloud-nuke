package aws

import (
	"fmt"
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
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

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

	region, err := getRandomRegion()
	require.NoError(t, err)

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

	region, err := getRandomRegion()
	require.NoError(t, err)

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

func TestNukeAccessAnalyzerMoreThanOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := accessanalyzer.New(session)

	analyzerNames := []*string{}
	for i := 0; i < 3; i++ {
		// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
		analyzerName := createAccessAnalyzer(t, svc)
		defer deleteAccessAnalyzer(t, svc, analyzerName, false)
		analyzerNames = append(analyzerNames, analyzerName)
	}

	require.NoError(
		t,
		nukeAllAccessAnalyzers(session, analyzerNames),
	)

	// Make sure the Access Analyzers are deleted.
	assertAccessAnalyzersDeleted(t, svc, analyzerNames)
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

	// AccessAnalyzer API operates on the name, so we need to extract out the name part of the ARN.
	arn, err := arn.Parse(aws.StringValue(resp.Arn))
	require.NoError(t, err)
	return arn.Resource
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
