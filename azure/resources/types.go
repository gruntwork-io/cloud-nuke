package resources

import (
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// AzureResource is the single canonical interface for all Azure resources. The
// azure package references this type for its container types and its registry.
type AzureResource interface {
	resource.NukeableResource
	Init(cfg AzureConfig)

	// LocationScope says how this resource type's location is decided, so a
	// reader can tell where the --region filter is applied. The one type Azure
	// registers reports its location on the API's answer, so its own lister
	// filters, and a type that is added with an identifier carrying its
	// location says so here rather than leaving the question to whoever reads
	// the lister. GCP learned that the expensive way: one of its four listers
	// consulted the filter and the other three deleted project-wide under it.
	LocationScope() LocationScope
}

// AzureResources holds multiple instances of AzureResource.
type AzureResources struct {
	Resources []*AzureResource
}

// AzureSubscriptionResources represents the resources found in one subscription.
type AzureSubscriptionResources struct {
	Resources map[string]AzureResources
}

// GetSubscription returns what was found in one subscription.
func (a *AzureSubscriptionResources) GetSubscription(subscriptionID string) AzureResources {
	if found, ok := a.Resources[subscriptionID]; ok {
		return found
	}
	return AzureResources{}
}

// TotalResourceCount returns how many individual resources were found.
func (a *AzureSubscriptionResources) TotalResourceCount() int {
	total := 0
	for _, inSubscription := range a.Resources {
		for _, r := range inSubscription.Resources {
			total += len((*r).ResourceIdentifiers())
		}
	}
	return total
}

// MapResourceTypeToIdentifiers maps each resource type to the identifiers found.
func (a *AzureSubscriptionResources) MapResourceTypeToIdentifiers() map[string][]string {
	identifiers := map[string][]string{}
	for _, inSubscription := range a.Resources {
		for _, r := range inSubscription.Resources {
			resourceType := (*r).ResourceName()
			identifiers[resourceType] = append(identifiers[resourceType], (*r).ResourceIdentifiers()...)
		}
	}
	return identifiers
}
