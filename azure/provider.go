package azure

import (
	"context"

	"github.com/gruntwork-io/cloud-nuke/azure/resources"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/collections"
)

// provider adapts Azure to the provider-neutral engine in the resource package.
// It holds only what the engine cannot know: which subscriptions to walk, how
// to build clients for one, and which failures are routine or a rate limit.
type provider struct {
	subscriptionIDs  []string
	locations        []string
	excludeLocations []string
	resourceTypes    []string
}

// Targets returns one target per subscription. A resource group is
// subscription-scoped and carries a location, so a location is a filter rather
// than a target: ARM would answer the same resources either way, and walking
// Azure's locations would multiply the API calls for nothing.
func (p *provider) Targets() ([]resource.Target, error) {
	targets := make([]resource.Target, 0, len(p.subscriptionIDs))
	for _, subscriptionID := range p.subscriptionIDs {
		targets = append(targets, resource.Target{ID: subscriptionID, Label: subscriptionID})
	}
	return targets, nil
}

// Prepare builds the Azure clients for one subscription and hands back every
// registered resource in registry order.
func (p *provider) Prepare(ctx context.Context, target resource.Target) (context.Context, []resource.NukeableResource, error) {
	registered := GetAndInitRegisteredResources(p.configFor(target))
	nukeable := make([]resource.NukeableResource, len(registered))
	for i := range registered {
		nukeable[i] = *registered[i]
	}
	return ctx, nukeable, nil
}

// configFor is what every resource in this target is initialized with: the
// subscription to walk and the location filter the engineer asked for. It is
// its own function so a test can read it without a credential.
func (p *provider) configFor(target resource.Target) resources.AzureConfig {
	return resources.AzureConfig{
		SubscriptionID:   target.ID,
		Locations:        p.locations,
		ExcludeLocations: p.excludeLocations,
	}
}

// SkipListError passes over nothing yet. Azure answers a provider that is not
// registered in the subscription with a 409 rather than an error that says the
// API is off, and no registered resource type can hit that, so every listing
// failure is reported.
func (p *provider) SkipListError(err error, resourceType string, explicitlyRequested bool) bool {
	return false
}

// IsThrottle reports whether the error is an ARM rate limit.
func (p *provider) IsThrottle(err error) bool {
	return isThrottlingError(err)
}

// explicitlyRequested reports whether the user named this resource type.
func (p *provider) explicitlyRequested(resourceType string) bool {
	return collections.ListContainsElement(p.resourceTypes, resourceType)
}

var _ resource.Provider = (*provider)(nil)
