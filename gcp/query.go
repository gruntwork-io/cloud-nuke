package gcp

import (
	"fmt"
	"strings"
	"time"
)

// Query represents the desired parameters for scanning GCP resources.
type Query struct {
	ProjectID            string
	ResourceTypes        []string
	ExcludeResourceTypes []string

	// Locations and ExcludeLocations come from --region and --exclude-region.
	// They narrow what each resource keeps while listing; they do not decide
	// what gets scanned, because every GCP list call is project-scoped.
	Locations        []string
	ExcludeLocations []string

	ExcludeAfter     *time.Time
	IncludeAfter     *time.Time
	Timeout          *time.Duration
	ExcludeFirstSeen bool
	Parallelism      int
}

// Validate ensures the query is usable before anything is scanned.
func (q *Query) Validate() error {
	if q.ProjectID == "" {
		return fmt.Errorf("--project-id is required")
	}

	if q.Parallelism < 0 {
		return fmt.Errorf("--parallelism must be >= 0 (0 uses the default)")
	}

	// "global" is an AWS pseudo-region, and GCP has no location by that name:
	// cloud-nuke used to require it here, so a runbook carrying it over would
	// now filter every resource out and report a clean project.
	for _, location := range append(append([]string{}, q.Locations...), q.ExcludeLocations...) {
		if strings.TrimSpace(location) == "" {
			return fmt.Errorf("--region and --exclude-region need a GCP location such as us-central1, and one was empty")
		}
		if strings.EqualFold(location, "global") {
			return fmt.Errorf("GCP has no \"global\" location: pass a GCP location such as us-central1, or drop --region and --exclude-region to scan the whole project")
		}
	}

	// An include list emptied by its own exclusions would scan everything,
	// which is the opposite of what the user asked for.
	if len(q.Locations) > 0 {
		var kept []string
		for _, location := range q.Locations {
			if !containsFold(q.ExcludeLocations, location) {
				kept = append(kept, location)
			}
		}
		if len(kept) == 0 {
			return fmt.Errorf("every --region was excluded by --exclude-region; nothing to scan")
		}
		q.Locations = kept
	}

	return nil
}
