package resource

import (
	"github.com/gruntwork-io/go-commons/collections"
)

// ResourceTypeLister is an interface for listing available resource types
type ResourceTypeLister interface {
	ListResourceTypes() []string
}

// HandleResourceTypeSelections accepts a slice of target resourceTypes and a slice of resourceTypes to exclude.
// It filters any excluded or invalid types from target resourceTypes then returns the filtered slice.
// This function is cloud-agnostic and can be used for both AWS and GCP resource type filtering.
func HandleResourceTypeSelections(
	includeResourceTypes, excludeResourceTypes []string,
	allResourceTypes []string,
) ([]string, error) {
	if len(includeResourceTypes) > 0 && len(excludeResourceTypes) > 0 {
		return []string{}, ResourceTypeAndExcludeFlagsBothPassedError{}
	}

	if len(includeResourceTypes) > 0 {
		return ensureValidResourceTypes(includeResourceTypes, allResourceTypes)
	}

	// Handle exclude resource types by going through the list of all types and only include those that are not
	// mentioned in the exclude list.
	validExcludeResourceTypes, err := ensureValidResourceTypes(excludeResourceTypes, allResourceTypes)
	if err != nil {
		return []string{}, err
	}

	resourceTypes := []string{}
	for _, resourceType := range allResourceTypes {
		if !collections.ListContainsElement(validExcludeResourceTypes, resourceType) {
			resourceTypes = append(resourceTypes, resourceType)
		}
	}
	return resourceTypes, nil
}

// ensureValidResourceTypes validates that all provided resource types are valid
func ensureValidResourceTypes(resourceTypes []string, allResourceTypes []string) ([]string, error) {
	invalidResourceTypes := []string{}
	for _, resourceType := range resourceTypes {
		if resourceType == "all" {
			continue
		}
		if !isValidResourceType(resourceType, allResourceTypes) {
			invalidResourceTypes = append(invalidResourceTypes, resourceType)
		}
	}

	if len(invalidResourceTypes) > 0 {
		return []string{}, InvalidResourceTypesSuppliedError{InvalidTypes: invalidResourceTypes}
	}

	return resourceTypes, nil
}

// isValidResourceType checks if a resourceType is valid
func isValidResourceType(resourceType string, allResourceTypes []string) bool {
	return collections.ListContainsElement(allResourceTypes, resourceType)
}
