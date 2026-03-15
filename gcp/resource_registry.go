package gcp

import (
	"github.com/gruntwork-io/cloud-nuke/gcp/resources"
)

// GlobalRegion is the region name used for GCP resources that are not region-scoped.
const GlobalRegion = "global"

// getRegisteredGlobalResources returns all GCP resource types that are global (not region-scoped).
func getRegisteredGlobalResources() []GcpResource {
	return []GcpResource{
		resources.NewGCSBuckets(),
		resources.NewCloudFunctions(),
		resources.NewArtifactRegistryRepositories(),
		resources.NewPubSubTopics(),
		resources.NewCloudRunServices(),
	}
}

// getRegisteredRegionalResources returns all GCP resource types that are region-scoped.
// Currently empty; regional resources will be added here as they are implemented.
func getRegisteredRegionalResources() []GcpResource {
	return []GcpResource{}
}

// GetAllRegisteredResources returns pointers to all registered GCP resources
// (both global and regional), without initializing them.
func GetAllRegisteredResources() []*GcpResource {
	all := append(getRegisteredGlobalResources(), getRegisteredRegionalResources()...)
	result := make([]*GcpResource, len(all))
	for i := range all {
		result[i] = &all[i]
	}
	return result
}

// GetAndInitRegisteredResources returns initialized GCP resources for the given region.
// Global resources are returned for GlobalRegion; regional resources for any other region.
func GetAndInitRegisteredResources(cfg resources.GcpConfig, region string) []*GcpResource {
	var raw []GcpResource
	if region == GlobalRegion {
		raw = getRegisteredGlobalResources()
	} else {
		raw = getRegisteredRegionalResources()
	}

	result := make([]*GcpResource, len(raw))
	for i := range raw {
		raw[i].Init(cfg)
		result[i] = &raw[i]
	}
	return result
}
