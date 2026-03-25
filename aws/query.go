package aws

import (
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
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
	ExcludeFirstSeen     bool
	DefaultOnly          bool
	IncludeTags          map[string]config.Expression
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
