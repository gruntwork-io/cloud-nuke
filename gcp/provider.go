package gcp

import (
	"context"
	"errors"

	"github.com/gruntwork-io/cloud-nuke/gcp/resources"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/collections"
)

// provider adapts GCP to the provider-neutral engine in the resource package.
//
// A GCP project is one target, not one target per location. Every list call GCP
// offers is project-scoped, and the ones that are not take a `locations/-`
// wildcard, so walking locations the way AWS walks regions would multiply the
// API calls without finding anything more. A location is therefore a filter a
// resource applies while it lists, which the engine never has to know about.
type provider struct {
	projectID        string
	locations        []string
	excludeLocations []string
	resourceTypes    []string
}

// Targets returns exactly one target: the project under which everything in GCP
// is listed. Its label is the project id, which is what the report shows.
func (p *provider) Targets() ([]resource.Target, error) {
	return []resource.Target{{ID: p.projectID, Label: p.projectID}}, nil
}

// Prepare builds the GCP clients for the project and hands back every
// registered resource in registry order.
func (p *provider) Prepare(ctx context.Context, target resource.Target) (context.Context, []resource.NukeableResource, error) {
	cfg := resources.GcpConfig{
		ProjectID:        target.ID,
		Locations:        p.locations,
		ExcludeLocations: p.excludeLocations,
	}

	registered := GetAndInitRegisteredResources(cfg)
	nukeable := make([]resource.NukeableResource, len(registered))
	for i := range registered {
		nukeable[i] = *registered[i]
	}
	return ctx, nukeable, nil
}

// SkipListError passes over a resource type whose API is switched off in this
// project, which is routine in GCP and not worth an error line, unless the user
// asked for that type by name and deserves to hear why it found nothing.
//
// A type that has no location to filter on is the same shape of answer: a run
// that asked for every type in us-central1 should not end in an error over
// Pub/Sub topics, which are in no location at all. It is skipped with one
// warning naming the type, so the report says nothing was deleted there and
// the run still exits clean. A run that named that type alongside --region
// asked for something contradictory, and keeps the error.
//
// This is the one place that knows both the error and whether the user asked
// for the type, which is why the warning is logged here rather than where the
// error is made.
func (p *provider) SkipListError(err error, resourceType string, explicitlyRequested bool) bool {
	var noLocation resources.LocationFilterNotApplicableError
	if errors.As(err, &noLocation) {
		if explicitlyRequested {
			return false
		}
		logging.Warnf("Leaving %s out of this run: it has no location, so --region cannot narrow it. Drop --region to include it.", resourceType)
		return true
	}

	return isServiceDisabledError(err) && !explicitlyRequested
}

// IsThrottle reports whether the error is a GCP quota or rate limit.
func (p *provider) IsThrottle(err error) bool {
	return isQuotaExhaustedError(err)
}

// explicitlyRequested reports whether the user named this resource type.
func (p *provider) explicitlyRequested(resourceType string) bool {
	return collections.ListContainsElement(p.resourceTypes, resourceType)
}

var _ resource.Provider = (*provider)(nil)
