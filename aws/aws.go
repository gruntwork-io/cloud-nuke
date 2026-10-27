package aws

import (
	"context"
	"fmt"
	"slices"

	"github.com/gruntwork-io/cloud-nuke/aws/resources"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/collections"
)

// GetAllResources - Lists all aws resources
func GetAllResources(c context.Context, query *Query, configObj config.Config, collector *reporting.Collector) (*AwsAccountResources, error) {
	configObj.AddExcludeAfterTime(query.ExcludeAfter)
	configObj.AddIncludeAfterTime(query.IncludeAfter)
	configObj.AddIncludeTags(query.IncludeTags)
	configObj.AddTimeout(query.Timeout)
	// Only override the config file value if the CLI flag is explicitly set to true
	if query.ListUnaliasedKMSKeys {
		configObj.KMSCustomerKeys.IncludeUnaliasedKeys = true
	}

	// Setting the DefaultOnly field
	// This function only sets the objects that have the `DefaultOnly` field, currently VPC, Subnet, and Security Group.
	configObj.AddEC2DefaultOnly(query.DefaultOnly)

	c = context.WithValue(c, util.ExcludeFirstSeenTagKey, query.ExcludeFirstSeen)

	found, err := resource.Scan(c, &provider{regions: query.Regions}, resource.ScanRequest{
		Include:     func(resourceType string) bool { return IsNukeable(resourceType, query.ResourceTypes) },
		Config:      configObj,
		Parallelism: query.Parallelism,
	}, collector)
	if err != nil {
		return nil, err
	}

	return accountResourcesFrom(found)
}

// NukeAllResources - Nukes all aws resources
func NukeAllResources(ctx context.Context, account *AwsAccountResources, regions []string, parallelism int, collector *reporting.Collector) error {
	return resource.NukeAll(ctx, &provider{regions: regions}, foundFrom(account, regions), resource.NukeRequest{Parallelism: parallelism}, collector)
}

// accountResourcesFrom converts what the engine found into the account-shaped
// value this package has always returned, which the CLI and library callers use.
// A resource that does not satisfy AwsResource is a programming error in the
// registry, and dropping it quietly would mean the engine found something that
// nothing then deletes.
func accountResourcesFrom(found *resource.Found) (*AwsAccountResources, error) {
	account := AwsAccountResources{Resources: make(map[string]AwsResources, len(found.Resources))}
	for region, nukeables := range found.Resources {
		awsResources := AwsResources{Resources: make([]*resources.AwsResource, 0, len(nukeables))}
		for _, nukeable := range nukeables {
			awsResource, ok := nukeable.(resources.AwsResource)
			if !ok {
				return nil, fmt.Errorf("%s in %s does not implement AwsResource", nukeable.ResourceName(), region)
			}
			awsResources.Resources = append(awsResources.Resources, &awsResource)
		}
		account.Resources[region] = awsResources
	}
	return &account, nil
}

// foundFrom converts an account-shaped value back for the engine, so a caller
// that scanned and nuked in two steps keeps working.
func foundFrom(account *AwsAccountResources, regions []string) *resource.Found {
	regions = uniqueRegions(regions)
	found := &resource.Found{Resources: make(map[string][]resource.NukeableResource, len(account.Resources))}
	for _, region := range regions {
		found.Targets = append(found.Targets, resource.Target{ID: region, Last: region == GlobalRegion})
	}
	for region, awsResources := range account.Resources {
		nukeables := make([]resource.NukeableResource, 0, len(awsResources.Resources))
		for _, awsResource := range awsResources.Resources {
			nukeables = append(nukeables, *awsResource)
		}
		found.Resources[region] = nukeables
	}
	return found
}

// ListResourceTypes - Returns list of resources which can be passed to --resource-type
func ListResourceTypes() []string {
	resourceTypes := []string{}
	for _, resource := range GetAllRegisteredResources() {
		resourceTypes = append(resourceTypes, (*resource).ResourceName())
	}

	slices.Sort(resourceTypes)
	return resourceTypes
}

// IsValidResourceType - Checks if a resourceType is valid or not
func IsValidResourceType(resourceType string, allResourceTypes []string) bool {
	return collections.ListContainsElement(allResourceTypes, resourceType)
}

// IsNukeable - Checks if we should nuke a resource or not
func IsNukeable(resourceType string, resourceTypes []string) bool {
	if len(resourceTypes) == 0 ||
		collections.ListContainsElement(resourceTypes, "all") ||
		collections.ListContainsElement(resourceTypes, resourceType) {
		return true
	}
	return false
}
