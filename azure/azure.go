package azure

import (
	"context"
	"fmt"
	"slices"

	"github.com/gruntwork-io/cloud-nuke/azure/resources"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/collections"
)

// AzureResources and AzureSubscriptionResources are exposed at the azure
// package level so a caller does not reach into azure/resources, which keeps
// the definitions where they are to avoid an import cycle.
type (
	AzureResource              = resources.AzureResource
	AzureResources             = resources.AzureResources
	AzureSubscriptionResources = resources.AzureSubscriptionResources
	AzureConfig                = resources.AzureConfig
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

// GetAllResources lists all Azure resources that can be deleted.
func GetAllResources(ctx context.Context, query *Query, configObj config.Config, collector *reporting.Collector) (*AzureSubscriptionResources, error) {
	configObj.AddExcludeAfterTime(query.ExcludeAfter)
	configObj.AddIncludeAfterTime(query.IncludeAfter)
	configObj.AddTimeout(query.Timeout)

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

	return subscriptionResourcesFrom(found)
}

// NukeSubscriptionResources nukes everything a scan found.
func NukeSubscriptionResources(ctx context.Context, account *AzureSubscriptionResources, subscriptionIDs []string, parallelism int, collector *reporting.Collector) error {
	return resource.NukeAll(ctx, &provider{subscriptionIDs: subscriptionIDs},
		foundFrom(account, subscriptionIDs), resource.NukeRequest{Parallelism: parallelism}, collector)
}

// newProvider builds the provider the engine walks for this query.
func newProvider(query *Query) *provider {
	return &provider{
		subscriptionIDs:  query.SubscriptionIDs,
		locations:        query.Locations,
		excludeLocations: query.ExcludeLocations,
		resourceTypes:    query.ResourceTypes,
	}
}

// subscriptionResourcesFrom converts what the engine found into the
// subscription-shaped value this package returns. A resource that does not
// satisfy AzureResource is a programming error in the registry, and dropping it
// quietly would mean the engine found something that nothing then deletes.
func subscriptionResourcesFrom(found *resource.Found) (*AzureSubscriptionResources, error) {
	subscriptions := AzureSubscriptionResources{Resources: make(map[string]AzureResources, len(found.Resources))}
	for id, nukeables := range found.Resources {
		azureResources := AzureResources{Resources: make([]*AzureResource, 0, len(nukeables))}
		for _, nukeable := range nukeables {
			azureResource, ok := nukeable.(AzureResource)
			if !ok {
				return nil, fmt.Errorf("%s in %s does not implement AzureResource", nukeable.ResourceName(), id)
			}
			azureResources.Resources = append(azureResources.Resources, &azureResource)
		}
		subscriptions.Resources[id] = azureResources
	}
	return &subscriptions, nil
}

// foundFrom converts a subscription-shaped value back for the engine.
func foundFrom(account *AzureSubscriptionResources, subscriptionIDs []string) *resource.Found {
	found := &resource.Found{Resources: make(map[string][]resource.NukeableResource, len(account.Resources))}
	for _, subscriptionID := range subscriptionIDs {
		found.Targets = append(found.Targets, resource.Target{ID: subscriptionID, Label: subscriptionID})
	}
	for id, azureResources := range account.Resources {
		nukeables := make([]resource.NukeableResource, 0, len(azureResources.Resources))
		for _, azureResource := range azureResources.Resources {
			nukeables = append(nukeables, *azureResource)
		}
		found.Resources[id] = nukeables
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
