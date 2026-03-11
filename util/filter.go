package util

import (
	"fmt"
	"strings"

	"github.com/gruntwork-io/go-commons/collections"
)

// InvalidResourceTypesSuppliedError is returned when invalid resource type names are provided.
type InvalidResourceTypesSuppliedError struct {
	InvalidTypes []string
}

func (err InvalidResourceTypesSuppliedError) Error() string {
	return fmt.Sprintf("Invalid resource types %v specified: Try --list-resource-types to get a list of valid resource types.", err.InvalidTypes)
}

// ResourceTypeAndExcludeFlagsBothPassedError is returned when both
// --resource-type and --exclude-resource-type are specified.
type ResourceTypeAndExcludeFlagsBothPassedError struct{}

func (err ResourceTypeAndExcludeFlagsBothPassedError) Error() string {
	return "You can not specify both --resource-type and --exclude-resource-type"
}

// IsNukeable checks if a resource type should be processed given the selected resource types.
// A nil slice means no filter was applied — all types are nukeable.
// A non-nil empty slice means all types were excluded — nothing is nukeable.
// Otherwise, returns true if the slice contains "all" (case-insensitive) or the specific type.
func IsNukeable(resourceType string, resourceTypes []string) bool {
	if resourceTypes == nil {
		return true
	}
	for _, rt := range resourceTypes {
		if strings.EqualFold(rt, "all") || rt == resourceType {
			return true
		}
	}
	return false
}

// IsValidResourceType checks if the given resource type exists in the provided list of valid types.
func IsValidResourceType(resourceType string, allResourceTypes []string) bool {
	return collections.ListContainsElement(allResourceTypes, resourceType)
}

// EnsureValidResourceTypes validates that all supplied resource types are valid.
// The "all" keyword (case-insensitive) is allowed and passes validation.
// Returns the validated slice or an error listing any invalid types.
func EnsureValidResourceTypes(resourceTypes, allResourceTypes []string) ([]string, error) {
	var invalidTypes []string
	for _, resourceType := range resourceTypes {
		if strings.EqualFold(resourceType, "all") {
			continue
		}
		if !IsValidResourceType(resourceType, allResourceTypes) {
			invalidTypes = append(invalidTypes, resourceType)
		}
	}

	if len(invalidTypes) > 0 {
		return []string{}, InvalidResourceTypesSuppliedError{InvalidTypes: invalidTypes}
	}

	return resourceTypes, nil
}

// HandleResourceTypeSelections validates and resolves include/exclude resource type flags
// into a final list of resource types to process. The two flags are mutually exclusive.
// A nil return slice signals "all types" (no filter applied).
// A non-nil empty return slice signals "no types" (all were excluded).
// Callers should use IsNukeable to check membership.
// The allResourceTypes parameter provides the full list of valid resource types.
func HandleResourceTypeSelections(includeResourceTypes, excludeResourceTypes, allResourceTypes []string) ([]string, error) {
	if len(includeResourceTypes) > 0 && len(excludeResourceTypes) > 0 {
		return []string{}, ResourceTypeAndExcludeFlagsBothPassedError{}
	}

	// No filters specified — return nil; IsNukeable treats nil as "all"
	if len(includeResourceTypes) == 0 && len(excludeResourceTypes) == 0 {
		return nil, nil
	}

	if len(includeResourceTypes) > 0 {
		return EnsureValidResourceTypes(includeResourceTypes, allResourceTypes)
	}

	// Reject "all" in exclude list — it is only meaningful as an include keyword
	for _, rt := range excludeResourceTypes {
		if strings.EqualFold(rt, "all") {
			return []string{}, InvalidResourceTypesSuppliedError{InvalidTypes: []string{rt}}
		}
	}

	validExcludeResourceTypes, err := EnsureValidResourceTypes(excludeResourceTypes, allResourceTypes)
	if err != nil {
		return []string{}, err
	}

	// Compute complement: all types minus excluded types.
	// Use make to ensure non-nil slice even when empty, so IsNukeable
	// distinguishes "no types selected" from "no filter applied" (nil).
	resourceTypes := make([]string, 0)
	for _, resourceType := range allResourceTypes {
		if !collections.ListContainsElement(validExcludeResourceTypes, resourceType) {
			resourceTypes = append(resourceTypes, resourceType)
		}
	}
	return resourceTypes, nil
}
