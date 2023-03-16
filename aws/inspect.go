package aws

import (
	"fmt"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/ui"
	"github.com/gruntwork-io/go-commons/collections"
)

// ExtractResourcesForPrinting is a convenience method that converts the nested structure of AwsAccountResources
// into a flat slice of resource identifiers, well-suited for printing line by line
func ExtractResourcesForPrinting(account *AwsAccountResources) []string {
	var resources []string

	if len(account.Resources) == 0 {
		logging.Logger.Infoln("No resources found!")
		return resources
	}

	resources = make([]string, 0)
	for region, resourcesInRegion := range account.Resources {
		for _, foundResources := range resourcesInRegion.Resources {
			for _, identifier := range foundResources.ResourceIdentifiers() {
				resources = append(resources, fmt.Sprintf("%s %s %s\n", ui.ResourceHighlightStyle.Render(foundResources.ResourceName()), identifier, region))
			}
		}
	}

	return resources
}

func ensureValidResourceTypes(resourceTypes []string) ([]string, error) {
	invalidresourceTypes := []string{}
	for _, resourceType := range resourceTypes {
		if resourceType == "all" {
			continue
		}
		if !IsValidResourceType(resourceType, ListResourceTypes()) {
			invalidresourceTypes = append(invalidresourceTypes, resourceType)
		}
	}

	if len(invalidresourceTypes) > 0 {
		return []string{}, InvalidResourceTypesSuppliedError{InvalidTypes: invalidresourceTypes}
	}

	return resourceTypes, nil
}

// HandleResourceTypeSelections accepts a slice of target resourceTypes and a slice of resourceTypes to exclude. It filters
// any excluded or invalid types from target resourceTypes then returns the filtered slice
func HandleResourceTypeSelections(
	includeResourceTypes, excludeResourceTypes []string,
) ([]string, error) {
	if len(includeResourceTypes) > 0 && len(excludeResourceTypes) > 0 {
		return []string{}, ResourceTypeAndExcludeFlagsBothPassedError{}
	}

	if len(includeResourceTypes) > 0 {
		return ensureValidResourceTypes(includeResourceTypes)
	}

	// Handle exclude resource types by going through the list of all types and only include those that are not
	// mentioned in the exclude list.
	validExcludeResourceTypes, err := ensureValidResourceTypes(excludeResourceTypes)
	if err != nil {
		return []string{}, err
	}

	resourceTypes := []string{}
	for _, resourceType := range ListResourceTypes() {
		if !collections.ListContainsElement(validExcludeResourceTypes, resourceType) {
			resourceTypes = append(resourceTypes, resourceType)
		}
	}
	return resourceTypes, nil
}

func InspectResources(q *Query) (*AwsAccountResources, error) {
	if len(q.ResourceTypes) > 0 {
		for _, resourceType := range q.ResourceTypes {
			logging.Logger.Infof("- %s", resourceType)
		}
	} else {
		for _, resourceType := range ListResourceTypes() {
			logging.Logger.Infof("- %s", resourceType)
		}
	}

	// NOTE: The inspect functionality currently does not support config file, so we short circuit the logic with an empty struct.
	return GetAllResources(q.Regions, q.ExcludeAfter, q.ResourceTypes, config.Config{})
}
