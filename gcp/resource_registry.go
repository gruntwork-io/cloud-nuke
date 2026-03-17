package gcp

import (
	"github.com/gruntwork-io/cloud-nuke/gcp/resources"
)

// getRegisteredResources returns all registered GCP resource types.
// Each resource is registered once — there is no global/regional split.
func getRegisteredResources() []GcpResource {
	return []GcpResource{
		resources.NewGCSBuckets(),
		resources.NewCloudFunctions(),
		resources.NewArtifactRegistryRepositories(),
		resources.NewPubSubTopics(),
		resources.NewSecretManagerSecrets(),
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

// GetAndInitRegisteredResources returns initialized GCP resources.
// Each resource is called exactly once; location filtering is handled by each resource's Lister.
func GetAndInitRegisteredResources(cfg resources.GcpConfig) []*GcpResource {
	raw := getRegisteredResources()

	result := make([]*GcpResource, len(raw))
	for i := range raw {
		raw[i].Init(cfg)
		result[i] = &raw[i]
	}
	return result
}
