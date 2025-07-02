package gcp

import (
	"context"

	"github.com/gruntwork-io/cloud-nuke/config"
)

// GcpResource is an interface that represents a single GCP resource
type GcpResource interface {
	Init(projectID string)
	ResourceName() string
	ResourceIdentifiers() []string
	MaxBatchSize() int
	Nuke(identifiers []string) error
	GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error)
	IsNukable(string) (bool, error)
	GetAndSetResourceConfig(config.Config) config.ResourceType
	PrepareContext(context.Context, config.ResourceType) error
}

// GcpResources is a struct to hold multiple instances of GcpResource.
type GcpResources struct {
	Resources []*GcpResource
}

// GcpProjectResources is a struct that represents the resources found in a single GCP project
type GcpProjectResources struct {
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
	for _, regionResource := range g.Resources {
		for _, resource := range regionResource.Resources {
			total += len((*resource).ResourceIdentifiers())
		}
	}
	return total
}

// MapResourceTypeToIdentifiers converts a slice of Resources to a map of resource types to their found identifiers
func (g *GcpProjectResources) MapResourceTypeToIdentifiers() map[string][]string {
	identifiers := map[string][]string{}
	for _, regionResource := range g.Resources {
		for _, resource := range regionResource.Resources {
			resourceType := (*resource).ResourceName()
			if _, ok := identifiers[resourceType]; !ok {
				identifiers[resourceType] = []string{}
			}
			identifiers[resourceType] = append(identifiers[resourceType], (*resource).ResourceIdentifiers()...)
		}
	}
	return identifiers
}
