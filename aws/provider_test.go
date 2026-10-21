package aws

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"

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
