package gcp

import (
	"fmt"

	"github.com/gruntwork-io/go-commons/collections"
)

// Query represents the desired parameters for scanning GCP resources.
// This mirrors the aws.Query struct for interface consistency.
type Query struct {
	ProjectID            string
	ResourceTypes        []string
	ExcludeResourceTypes []string
	Regions              []string
	ExcludeRegions       []string
}

// Validate ensures the query has valid defaults.
// If no regions are specified, it defaults to GlobalRegion.
// ExcludeRegions are filtered out from the region list.
// Validates that requested resource types exist.
func (q *Query) Validate() error {
	if len(q.Regions) == 0 {
		q.Regions = []string{GlobalRegion}
	}

	if len(q.ExcludeRegions) > 0 {
		var filtered []string
		for _, region := range q.Regions {
			if !collections.ListContainsElement(q.ExcludeRegions, region) {
				filtered = append(filtered, region)
			}
		}
		q.Regions = filtered
	}

	if len(q.Regions) == 0 {
		return fmt.Errorf("no regions to process after applying exclusions")
	}

	// Validate resource types and exclude resource types
	needsValidation := (len(q.ResourceTypes) > 0 && !collections.ListContainsElement(q.ResourceTypes, "all")) ||
		len(q.ExcludeResourceTypes) > 0
	if needsValidation {
		validTypes := ListResourceTypes()

		if len(q.ResourceTypes) > 0 && !collections.ListContainsElement(q.ResourceTypes, "all") {
			var invalidTypes []string
			for _, rt := range q.ResourceTypes {
				if !collections.ListContainsElement(validTypes, rt) {
					invalidTypes = append(invalidTypes, rt)
				}
			}
			if len(invalidTypes) > 0 {
				return fmt.Errorf("invalid resource type(s) %v specified. Use --list-resource-types to see valid types", invalidTypes)
			}
		}

		if len(q.ExcludeResourceTypes) > 0 {
			var invalidTypes []string
			for _, rt := range q.ExcludeResourceTypes {
				if !collections.ListContainsElement(validTypes, rt) {
					invalidTypes = append(invalidTypes, rt)
				}
			}
			if len(invalidTypes) > 0 {
				return fmt.Errorf("invalid exclude resource type(s) %v specified. Use --list-resource-types to see valid types", invalidTypes)
			}
		}
	}

	return nil
}
