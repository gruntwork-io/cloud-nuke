package gcp

import (
	"testing"

	"github.com/gruntwork-io/cloud-nuke/gcp/resources"
	"github.com/stretchr/testify/assert"
)

// TestEveryResourceDeclaresALocationScope pins how each registered resource
// type answers --region. A new resource type fails this test until somebody
// decides, which is what stops a lister that ignores the filter from widening
// a run that the user narrowed.
func TestEveryResourceDeclaresALocationScope(t *testing.T) {
	t.Parallel()

	expected := map[string]resources.LocationScope{
		// Identifiers are full resource names, so the adapter reads the
		// location out of the name and filters.
		"cloud-function":    resources.LocationFromResourceName,
		"artifact-registry": resources.LocationFromResourceName,

		// A bucket's name carries no location, so its lister filters, and a
		// Cloud SQL instance reports its region the same way.
		"gcs-bucket":         resources.LocationAppliedByLister,
		"cloud-sql-instance": resources.LocationAppliedByLister,

		// An instance lives in a zone, which is not what --region names, so
		// its lister matches a zone against the region it is in.
		"compute-instance": resources.LocationAppliedByLister,

		// A Pub/Sub topic is project-global, so a location filter cannot
		// include it and the resource refuses to run under one.
		"gcp-pubsub-topic": resources.LocationNone,
	}

	registered := map[string]resources.LocationScope{}
	for _, r := range GetAllRegisteredResources() {
		registered[(*r).ResourceName()] = (*r).LocationScope()
	}

	assert.Equal(t, expected, registered,
		"a registered GCP resource has no declared location scope, or its scope changed: decide how --region applies to it, then update this test")
}
