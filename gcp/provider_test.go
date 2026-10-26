package gcp

import (
	"context"
	"errors"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/gcp/resources"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkipListErrorLeavesOutALocationlessTypeButKeepsOneTheUserNamed(t *testing.T) {
	t.Parallel()

	noLocation := resources.LocationFilterNotApplicableError{ResourceTypeName: "pubsub-topic"}
	p := &provider{projectID: "p", locations: []string{"us-central1"}}

	// A run that asked for everything in one location should not end in an
	// error over a type that is in no location.
	assert.True(t, p.SkipListError(noLocation, "pubsub-topic", false))

	// Naming the type alongside --region is contradictory, so it is an error.
	assert.False(t, p.SkipListError(noLocation, "pubsub-topic", true))

	// A wrapped one is recognized too, since listers wrap what they return.
	assert.True(t, p.SkipListError(errors.Join(errors.New("listing failed"), noLocation), "pubsub-topic", false))

	// Anything else is still a failure worth reporting.
	assert.False(t, p.SkipListError(errors.New("permission denied"), "pubsub-topic", false))
}

// The filter the engineer typed has to reach the resources, and the path it
// takes is the query, the provider and the config Prepare builds. A type that
// is in no location is where that path shows: it is left out only when the
// config it was initialized with carries the locations.
func TestTheProviderCarriesTheQuerysLocationsIntoEveryResource(t *testing.T) {
	t.Parallel()

	query := &Query{ProjectID: "p", Locations: []string{"us-central1"}}
	require.NoError(t, query.Validate())

	p := newProvider(query)
	_, nukeables, err := p.Prepare(context.Background(), resource.Target{ID: "p", Label: "p"})
	require.NoError(t, err)
	require.NotEmpty(t, nukeables)

	var locationless resource.NukeableResource
	for _, r := range nukeables {
		if r.ResourceName() == "gcp-pubsub-topic" {
			locationless = r
		}
	}
	require.NotNil(t, locationless, "the registry holds a resource type that is in no location")

	_, err = locationless.GetAndSetIdentifiers(context.Background(), config.Config{})
	require.Error(t, err, "a locationless type under a location filter refuses to list")

	var notApplicable resources.LocationFilterNotApplicableError
	require.ErrorAs(t, err, &notApplicable, "the refusal is the typed one the provider leaves out of the run")
	assert.Equal(t, "gcp-pubsub-topic", notApplicable.ResourceTypeName)
}

func TestTheProviderKeepsADisabledApiFailureWhenTheUserNamedThatType(t *testing.T) {
	t.Parallel()

	// A disabled API is routine and skipped in silence, except for the one
	// case the parameter exists for: the engineer asked for that type by name
	// and would otherwise be told nothing at all.
	p := &provider{projectID: "p", resourceTypes: []string{"gcs-bucket"}}
	disabled := makeServiceDisabledErr()

	assert.True(t, p.SkipListError(disabled, "cloud-function", false))
	assert.False(t, p.SkipListError(disabled, "gcs-bucket", p.explicitlyRequested("gcs-bucket")))
}
