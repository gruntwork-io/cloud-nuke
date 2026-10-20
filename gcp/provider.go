package gcp

import (
	"context"

	"github.com/gruntwork-io/cloud-nuke/gcp/resources"
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
func (p *provider) SkipListError(err error, resourceType string, explicitlyRequested bool) bool {
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
