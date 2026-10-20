package azure

import (
	"context"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/gruntwork-io/cloud-nuke/azure/resources"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTargetsIsOneSubscriptionEach pins the decision that a location is a
// filter rather than a target: ARM list calls cover the whole subscription and
// report each resource's location, so walking Azure's locations would multiply
// the API calls and find the same resources.
func TestTargetsIsOneSubscriptionEach(t *testing.T) {
	t.Parallel()

	p := &provider{subscriptionIDs: []string{"sub-a", "sub-b"}}

	targets, err := p.Targets()
	require.NoError(t, err)
	require.Len(t, targets, 2)

	for _, target := range targets {
		assert.Equal(t, target.ID, target.Label)
		assert.False(t, target.Last, "no Azure target is deferred yet")
	}
	assert.Equal(t, "sub-a", targets[0].ID)
}

func TestPrepareGivesEachResourceTheSubscriptionAndTheLocationFilter(t *testing.T) {
	t.Parallel()

	p := &provider{
		subscriptionIDs: []string{"sub-a"},
		locations:       []string{"westeurope"},
	}

	_, nukeable, err := p.Prepare(context.Background(), resource.Target{ID: "sub-a"})
	require.NoError(t, err)
	require.NotEmpty(t, nukeable)

	// Every registered resource is handed back in registry order, and each one
	// declares how its location is decided.
	for _, r := range nukeable {
		azureResource, ok := r.(resources.AzureResource)
		require.True(t, ok)
		assert.Equal(t, resources.LocationReportedByLister, azureResource.LocationScope())
	}
}

func TestIsThrottleRecognisesWhatArmAnswersUnderLoad(t *testing.T) {
	t.Parallel()

	p := &provider{}

	assert.True(t, p.IsThrottle(&azcore.ResponseError{StatusCode: http.StatusTooManyRequests}))
	assert.True(t, p.IsThrottle(&azcore.ResponseError{StatusCode: http.StatusServiceUnavailable}))

	// A refusal is not a rate limit: waiting does not fix it, so the engine
	// must record it as a failure rather than sleeping.
	assert.False(t, p.IsThrottle(&azcore.ResponseError{StatusCode: http.StatusForbidden}))
	assert.False(t, p.IsThrottle(&azcore.ResponseError{StatusCode: http.StatusNotFound}))
	assert.False(t, p.IsThrottle(nil))
}

// TestEveryResourceDeclaresALocationScope pins how each registered resource
// answers --region. A new resource type fails this test until somebody decides,
// which is what stops a lister that ignores the filter from widening a run the
// user narrowed.
func TestEveryResourceDeclaresALocationScope(t *testing.T) {
	t.Parallel()

	expected := map[string]resources.LocationScope{
		// ARM reports a resource group's location on the list response.
		"resource-group": resources.LocationReportedByLister,
	}

	registered := map[string]resources.LocationScope{}
	for _, r := range GetAllRegisteredResources() {
		registered[(*r).ResourceName()] = (*r).LocationScope()
	}

	assert.Equal(t, expected, registered,
		"a registered Azure resource has no declared location scope, or its scope changed: decide how --region applies to it, then update this test")
}

func TestFoundRoundTripsThroughTheSubscriptionShape(t *testing.T) {
	t.Parallel()

	registered := GetAndInitRegisteredResources(resources.AzureConfig{SubscriptionID: "sub-a"})
	require.NotEmpty(t, registered)

	account := &AzureSubscriptionResources{Resources: map[string]AzureResources{
		"sub-a": {Resources: registered},
	}}

	found := foundFrom(account, []string{"sub-a"})
	require.Len(t, found.Targets, 1)
	require.Len(t, found.Resources["sub-a"], len(registered))

	back, err := subscriptionResourcesFrom(found)
	require.NoError(t, err)
	require.Len(t, back.Resources["sub-a"].Resources, len(registered))
	assert.Equal(t,
		(*registered[0]).ResourceName(),
		(*back.Resources["sub-a"].Resources[0]).ResourceName())
}
