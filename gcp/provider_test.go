package gcp

import (
	"errors"
	"testing"

	"github.com/gruntwork-io/cloud-nuke/gcp/resources"
	"github.com/stretchr/testify/assert"
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
