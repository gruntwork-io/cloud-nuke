package util

import "github.com/gruntwork-io/go-commons/collections"

// IsNukeable checks whether a resource type should be included based on include/exclude lists.
// An empty includeTypes list or the special value "all" means include everything.
func IsNukeable(resourceType string, includeTypes []string, excludeTypes []string) bool {
	if collections.ListContainsElement(excludeTypes, resourceType) {
		return false
	}
	if len(includeTypes) == 0 ||
		collections.ListContainsElement(includeTypes, "all") ||
		collections.ListContainsElement(includeTypes, resourceType) {
		return true
	}
	return false
}
