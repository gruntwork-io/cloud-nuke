package aws

import (
	"time"
)

// Query is a struct that represents the desired parameters for scanning resources within a given account
type Query struct {
	Regions              []string
	ExcludeRegions       []string
	ResourceTypes        []string
	ExcludeResourceTypes []string
	ExcludeAfter         *time.Time
	IncludeAfter         *time.Time
	ListUnaliasedKMSKeys bool
	Timeout              *time.Duration
}

// NewQuery configures and returns a Query struct that can be passed into the InspectResources method
func NewQuery(regions, excludeRegions, resourceTypes, excludeResourceTypes []string, excludeAfter, includeAfter *time.Time, listUnaliasedKMSKeys bool, timeout *time.Duration) (*Query, error) {
	q := &Query{
		Regions:              regions,
		ExcludeRegions:       excludeRegions,
		ResourceTypes:        resourceTypes,
		ExcludeResourceTypes: excludeResourceTypes,
		ExcludeAfter:         excludeAfter,
		IncludeAfter:         includeAfter,
		ListUnaliasedKMSKeys: listUnaliasedKMSKeys,
		Timeout:              timeout,
	}

	validationErr := q.Validate()

	if validationErr != nil {
		return q, validationErr
	}

	return q, nil
}

// Validate ensures the configured values for a Query are valid, returning an error if there are
// any invalid params, or nil if the Query is valid
func (q *Query) Validate() error {
	resourceTypes, err := HandleResourceTypeSelections(q.ResourceTypes, q.ExcludeResourceTypes)
	if err != nil {
		return err
	}

	q.ResourceTypes = resourceTypes

	regions, err := GetEnabledRegions()
	if err != nil {
		return CouldNotDetermineEnabledRegionsError{Underlying: err}
	}

	// global is a fake region, used to represent global resources
	regions = append(regions, GlobalRegion)

	targetRegions, err := GetTargetRegions(regions, q.Regions, q.ExcludeRegions)
	if err != nil {
		return CouldNotSelectRegionError{Underlying: err}
	}

	q.Regions = targetRegions

	return nil
}
