package resources

import (
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// GcpResource is an interface that represents a single GCP resource.
// This interface is satisfied by GcpResourceAdapter[C] which wraps resource.Resource[C].
type GcpResource interface {
	resource.NukeableResource
	Init(cfg GcpConfig)
}

// GcpResources is a struct to hold multiple instances of GcpResource.
type GcpResources struct {
	Resources []*GcpResource
}

// GcpProjectResources represents the resources found in a single GCP project.
// The map key is "project" (a single flat namespace, not keyed by region).
type GcpProjectResources struct {
	ProjectID string
	Resources map[string]GcpResources
}

func (g *GcpProjectResources) GetRegion(region string) GcpResources {
	if val, ok := g.Resources[region]; ok {
		return val
	}
	return GcpResources{}
}

// TotalResourceCount returns the number of resources found, that are eligible for nuking
func (g *GcpProjectResources) TotalResourceCount() int {
	total := 0
	for _, projectResource := range g.Resources {
		for _, resource := range projectResource.Resources {
			total += len((*resource).ResourceIdentifiers())
		}
	}
	return total
}

// MapResourceTypeToIdentifiers converts a slice of Resources to a map of resource types to their found identifiers
func (g *GcpProjectResources) MapResourceTypeToIdentifiers() map[string][]string {
	identifiers := map[string][]string{}
	for _, projectResource := range g.Resources {
		for _, resource := range projectResource.Resources {
			resourceType := (*resource).ResourceName()
			if _, ok := identifiers[resourceType]; !ok {
				identifiers[resourceType] = []string{}
			}
			identifiers[resourceType] = append(identifiers[resourceType], (*resource).ResourceIdentifiers()...)
		}
	}
	return identifiers
}
