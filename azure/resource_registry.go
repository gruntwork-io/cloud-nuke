package azure

import (
	"github.com/gruntwork-io/cloud-nuke/azure/resources"
)

// getRegisteredResources returns every Azure resource type, in the order they
// must be deleted. There is no location split: an ARM list call covers the
// whole subscription and reports each resource's location, so a location is a
// filter each lister applies rather than a scope to walk.
func getRegisteredResources() []resources.AzureResource {
	return []resources.AzureResource{
		// A resource group is deleted last, because Azure deletes everything
		// inside it with it: deleting one first would pull resources out from
		// under the types registered above.
		resources.NewResourceGroups(),
	}
}

// GetAllRegisteredResources returns pointers to all registered Azure resources
// without initializing them.
func GetAllRegisteredResources() []*resources.AzureResource {
	all := getRegisteredResources()

	result := make([]*resources.AzureResource, len(all))
	for i := range all {
		result[i] = &all[i]
	}
	return result
}

// GetAndInitRegisteredResources returns initialized Azure resources for a
// subscription.
func GetAndInitRegisteredResources(cfg resources.AzureConfig) []*resources.AzureResource {
	raw := getRegisteredResources()

	result := make([]*resources.AzureResource, len(raw))
	for i := range raw {
		raw[i].Init(cfg)
		result[i] = &raw[i]
	}
	return result
}
