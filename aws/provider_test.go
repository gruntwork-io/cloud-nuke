package aws

import (
	"errors"
	"fmt"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/smithy-go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTargetsDefersTheGlobalRegion pins the one thing that makes the engine's
// deferral mean anything for AWS: IAM, S3, CloudFront and Route53 deletions can
// break a regional delete that is still running, so the global pseudo-region
// goes last.
func TestTargetsDefersTheGlobalRegion(t *testing.T) {
	t.Parallel()

	p := &provider{regions: []string{"us-east-1", GlobalRegion, "eu-west-1"}}

	targets, err := p.Targets()
	require.NoError(t, err)
	require.Len(t, targets, 3)

	deferred := map[string]bool{}
	for _, target := range targets {
		deferred[target.ID] = target.Last
	}
	assert.Equal(t, map[string]bool{
		"us-east-1":  false,
		"eu-west-1":  false,
		GlobalRegion: true,
	}, deferred)
}

// TestFoundRoundTripsThroughTheAccountShape pins that the value this package
// has always returned survives the trip to the engine and back. Dropping a
// resource in either direction is a nuke that deletes nothing and reports
// success.
func TestFoundRoundTripsThroughTheAccountShape(t *testing.T) {
	t.Parallel()

	regions := []string{"us-east-1", GlobalRegion}
	registered := GetAndInitRegisteredResources(awsConfigForTest(), "us-east-1")
	require.NotEmpty(t, registered)

	// One resource type found in one region, which is the shape a scan returns.
	account := &AwsAccountResources{Resources: map[string]AwsResources{
		"us-east-1": {Resources: registered[:1]},
	}}

	found := foundFrom(account, regions)
	require.Len(t, found.Targets, 2)
	require.Len(t, found.Resources["us-east-1"], 1)

	back, err := accountResourcesFrom(found)
	require.NoError(t, err)
	require.Len(t, back.Resources["us-east-1"].Resources, 1)
	assert.Equal(t,
		(*registered[0]).ResourceName(),
		(*back.Resources["us-east-1"].Resources[0]).ResourceName())
}

// awsConfigForTest builds a config with no credentials, which is enough to
// construct clients without calling an API.
func awsConfigForTest() awssdk.Config {
	return awssdk.Config{Region: "us-east-1"}
}

// A region named twice is one region: the engine refuses a repeated target
// rather than deleting it twice, so a scripted region list that holds a
// duplicate narrows here instead of ending the run.
func TestTargetsNamesEachRegionOnce(t *testing.T) {
	t.Parallel()

	p := &provider{regions: []string{"us-east-1", "eu-west-1", "us-east-1", GlobalRegion}}

	targets, err := p.Targets()
	require.NoError(t, err)

	ids := make([]string, 0, len(targets))
	for _, target := range targets {
		ids = append(ids, target.ID)
	}
	assert.Equal(t, []string{"us-east-1", "eu-west-1", GlobalRegion}, ids)
}

// AWS skips no listing failure. Every one reaches the report, because a region
// that answered nothing is a region whose resources are still there.
func TestSkipListErrorKeepsEveryAwsListingFailure(t *testing.T) {
	t.Parallel()

	p := &provider{}

	assert.False(t, p.SkipListError(errors.New("AccessDenied"), "ec2", false))
	assert.False(t, p.SkipListError(errors.New("AccessDenied"), "ec2", true))
	assert.False(t, p.SkipListError(&smithy.GenericAPIError{Code: "ThrottlingException"}, "ec2", false))
}

// The engine waits out a rate limit rather than recording it as a failure, and
// this is the answer that tells it which is which.
func TestIsThrottleRecognizesTheRateLimitsAwsAnswersWith(t *testing.T) {
	t.Parallel()

	p := &provider{}

	for _, code := range []string{"RequestLimitExceeded", "ThrottlingException", "TooManyRequestsException"} {
		assert.True(t, p.IsThrottle(&smithy.GenericAPIError{Code: code}), code)
		assert.True(t, p.IsThrottle(fmt.Errorf("deleting: %w", &smithy.GenericAPIError{Code: code})), "wrapped "+code)
	}

	assert.False(t, p.IsThrottle(&smithy.GenericAPIError{Code: "AccessDenied"}))
	assert.False(t, p.IsThrottle(errors.New("dial tcp: timeout")))
}
