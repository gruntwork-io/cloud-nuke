package gcp

import (
	"fmt"
	"time"

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
	ExcludeAfter         *time.Time
	IncludeAfter         *time.Time
	Timeout              *time.Duration
	ExcludeFirstSeen     bool
}

// Validate ensures the query has valid defaults.
// If no regions are specified, it defaults to GlobalRegion.
// ExcludeRegions are filtered out from the region list.
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

	return nil
}
