package gcp

import (
	"github.com/gruntwork-io/cloud-nuke/gcp/resources"
)

// getRegisteredResources returns every GCP resource type, in the order they
// must be deleted. There is no global and regional split: a GCP project is
// scanned in one pass and a location is a filter each lister applies.
func getRegisteredResources() []GcpResource {
	return []GcpResource{
		// A virtual machine holds the disks and the network interfaces a later
		// resource type deletes, so it goes first.
		resources.NewComputeInstances(),
		resources.NewGCSBuckets(),
		resources.NewCloudFunctions(),
		resources.NewArtifactRegistryRepositories(),
		resources.NewPubSubTopics(),
	}
}

// GetAllRegisteredResources returns pointers to all registered GCP resources
// without initializing them.
func GetAllRegisteredResources() []*GcpResource {
	all := getRegisteredResources()
	result := make([]*GcpResource, len(all))
	for i := range all {
		result[i] = &all[i]
	}
	return result
}

// GetAndInitRegisteredResources returns initialized GCP resources for a project.
func GetAndInitRegisteredResources(cfg resources.GcpConfig) []*GcpResource {
	raw := getRegisteredResources()

	result := make([]*GcpResource, len(raw))
	for i := range raw {
		raw[i].Init(cfg)
		result[i] = &raw[i]
	}
	return result
}
