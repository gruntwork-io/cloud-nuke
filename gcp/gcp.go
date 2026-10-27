package gcp

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/collections"
)

// IsNukeable checks whether a resource type should be nuked based on the
// requested resource types and exclude lists. An empty include list or the
// special value "all" means nuke everything, minus any excluded types.
func IsNukeable(resourceType string, resourceTypes []string, excludeResourceTypes []string) bool {
	if collections.ListContainsElement(excludeResourceTypes, resourceType) {
		return false
	}
	if len(resourceTypes) == 0 ||
		collections.ListContainsElement(resourceTypes, "all") ||
		collections.ListContainsElement(resourceTypes, resourceType) {
		return true
	}
	return false
}

// GetAllResources lists all GCP resources that can be deleted.
func GetAllResources(ctx context.Context, query *Query, configObj config.Config, collector *reporting.Collector) (*GcpProjectResources, error) {
	ctx = context.WithValue(ctx, util.ExcludeFirstSeenTagKey, query.ExcludeFirstSeen)

	p := newProvider(query)
	found, err := resource.Scan(ctx, p, resource.ScanRequest{
		Include: func(resourceType string) bool {
			return IsNukeable(resourceType, query.ResourceTypes, query.ExcludeResourceTypes)
		},
		ExplicitlyRequested: p.explicitlyRequested,
		Config:              configObj,
		Parallelism:         query.Parallelism,
	}, collector)
	if err != nil {
		return nil, err
	}

	return projectResourcesFrom(found)
}

// NukeProjectResources nukes everything a scan found in the project.
//
// This replaces NukeAllResources, which took a slice of regions. The name
// changes with the meaning: keeping it would let a caller pass regions[0] as
// the project id, compile, and nuke a project named after a region.
func NukeProjectResources(ctx context.Context, account *GcpProjectResources, projectID string, parallelism int, collector *reporting.Collector) error {
	return resource.NukeAll(ctx, &provider{projectID: projectID}, foundFrom(account, projectID), resource.NukeRequest{Parallelism: parallelism}, collector)
}

// newProvider builds the provider the engine walks for this query.
func newProvider(query *Query) *provider {
	return &provider{
		projectID:        query.ProjectID,
		locations:        query.Locations,
		excludeLocations: query.ExcludeLocations,
		resourceTypes:    query.ResourceTypes,
	}
}

// projectResourcesFrom converts what the engine found into the project-shaped
// value this package has always returned. A resource that does not satisfy
// GcpResource is a programming error in the registry, and dropping it quietly
// would mean the engine found something that nothing then deletes.
func projectResourcesFrom(found *resource.Found) (*GcpProjectResources, error) {
	project := GcpProjectResources{Resources: make(map[string]GcpResources, len(found.Resources))}
	for id, nukeables := range found.Resources {
		gcpResources := GcpResources{Resources: make([]*GcpResource, 0, len(nukeables))}
		for _, nukeable := range nukeables {
			gcpResource, ok := nukeable.(GcpResource)
			if !ok {
				return nil, fmt.Errorf("%s in %s does not implement GcpResource", nukeable.ResourceName(), id)
			}
			gcpResources.Resources = append(gcpResources.Resources, &gcpResource)
		}
		project.Resources[id] = gcpResources
	}
	return &project, nil
}

// foundFrom converts a project-shaped value back for the engine.
func foundFrom(account *GcpProjectResources, projectID string) *resource.Found {
	found := &resource.Found{
		Targets:   []resource.Target{{ID: projectID, Label: projectID}},
		Resources: make(map[string][]resource.NukeableResource, len(account.Resources)),
	}
	for label, gcpResources := range account.Resources {
		nukeables := make([]resource.NukeableResource, 0, len(gcpResources.Resources))
		for _, gcpResource := range gcpResources.Resources {
			nukeables = append(nukeables, *gcpResource)
		}
		found.Resources[label] = nukeables
	}
	return found
}

// ListResourceTypes returns a sorted list of resources which can be passed to --resource-type
func ListResourceTypes() []string {
	var resourceTypes []string
	for _, r := range GetAllRegisteredResources() {
		resourceTypes = append(resourceTypes, (*r).ResourceName())
	}
	slices.Sort(resourceTypes)
	return resourceTypes
}

// containsFold reports whether the list holds the target, ignoring case, since
// GCP location names are written both ways in the wild.
func containsFold(list []string, target string) bool {
	for _, item := range list {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}
