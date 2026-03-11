package aws

import (
	"github.com/gruntwork-io/cloud-nuke/util"
)

// HandleResourceTypeSelections accepts a slice of target resourceTypes and a slice of resourceTypes to exclude. It filters
// any excluded or invalid types from target resourceTypes then returns the filtered slice.
// A nil return means "all types"; a non-nil empty return means "no types".
func HandleResourceTypeSelections(
	includeResourceTypes, excludeResourceTypes []string,
) ([]string, error) {
	return util.HandleResourceTypeSelections(includeResourceTypes, excludeResourceTypes, ListResourceTypes())
}
