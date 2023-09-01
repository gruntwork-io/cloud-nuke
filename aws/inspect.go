package aws

import (
	"github.com/gruntwork-io/go-commons/collections"
)

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
