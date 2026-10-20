package azure

import (
	"fmt"
	"strings"
	"time"
)

// Query represents the desired parameters for scanning Azure resources.
type Query struct {
	SubscriptionIDs      []string
	ResourceTypes        []string
	ExcludeResourceTypes []string

	// Locations and ExcludeLocations come from --region and --exclude-region.
	// They narrow what each resource keeps while listing; they do not decide
	// what gets scanned, because an ARM list call covers the subscription.
	Locations        []string
	ExcludeLocations []string

	ExcludeAfter *time.Time
	IncludeAfter *time.Time
	Timeout      *time.Duration
	Parallelism  int
}

// Validate ensures the query is usable before anything is scanned.
func (q *Query) Validate() error {
	if len(q.SubscriptionIDs) == 0 {
		return fmt.Errorf("--subscription-id is required")
	}

	seen := make(map[string]struct{}, len(q.SubscriptionIDs))
	for _, subscription := range q.SubscriptionIDs {
		if strings.TrimSpace(subscription) == "" {
			return fmt.Errorf("--subscription-id was empty")
		}
		if _, duplicate := seen[subscription]; duplicate {
			return fmt.Errorf("--subscription-id %s was given twice, which would scan and delete it twice", subscription)
		}
		seen[subscription] = struct{}{}
	}

	if q.Parallelism < 0 {
		return fmt.Errorf("--parallelism must be >= 0 (0 uses the default)")
	}

	for _, location := range append(append([]string{}, q.Locations...), q.ExcludeLocations...) {
		if strings.TrimSpace(location) == "" {
			return fmt.Errorf("--region and --exclude-region need an Azure location such as westeurope, and one was empty")
		}
		// "global" is an AWS pseudo-region. Azure reports a location of
		// "global" for a few subscription-level resources, but none of the
		// registered types use it, so accepting it would filter everything out.
		if strings.EqualFold(location, "global") {
			return fmt.Errorf("there is no \"global\" Azure location for the resource types cloud-nuke deletes: pass a location such as westeurope, or drop --region to scan the whole subscription")
		}
	}

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

// containsFold reports whether the list holds the target, ignoring case.
func containsFold(list []string, target string) bool {
	for _, item := range list {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}
