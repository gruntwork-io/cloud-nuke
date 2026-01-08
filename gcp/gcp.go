package gcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/gcp/resources"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"github.com/pterm/pterm"
)

// GetAllResources lists all GCP resources that can be deleted.
// The context can contain a reporting.Collector for event-driven reporting.
func GetAllResources(ctx context.Context, projectID string, configObj config.Config, excludeAfter time.Time, includeAfter time.Time) (*GcpProjectResources, error) {
	allResources := GcpProjectResources{
		Resources: map[string]GcpResources{},
	}

	// Get all resource types to delete
	resourceTypes := getAllResourceTypes()

	// Create a progress bar
	bar, _ := pterm.DefaultProgressbar.WithTotal(len(resourceTypes)).WithTitle("Retrieving GCP resources").Start()

	// Get collector from context if present
	collector := reporting.FromContext(ctx)

	// For each resource type
	for _, resourceType := range resourceTypes {
		// Update progress bar
		bar.UpdateTitle(fmt.Sprintf("Retrieving GCP %s", resourceType.ResourceName()))

		// Initialize the resource
		resourceType.Init(projectID)

		// Get all resource identifiers
		if _, err := resourceType.GetAndSetIdentifiers(ctx, configObj); err != nil {
			logging.Debugf("Error getting identifiers for %s: %v", resourceType.ResourceName(), err)
			// Report error to collector if present
			if collector != nil {
				collector.RecordError(resourceType.ResourceName(), fmt.Sprintf("Unable to retrieve %s", resourceType.ResourceName()), err)
			}
			continue
		}

		// Report found resources to collector if present (for inspect operations)
		if collector != nil {
			for _, id := range resourceType.ResourceIdentifiers() {
				nukable, reason := resourceType.IsNukable(id)
				reasonStr := ""
				if reason != nil {
					reasonStr = reason.Error()
				}
				collector.RecordFound(resourceType.ResourceName(), "global", id, nukable, reasonStr)
			}
		}

		// Add the resource to the map
		allResources.Resources["global"] = GcpResources{
			Resources: append(allResources.Resources["global"].Resources, &resourceType),
		}

		// Increment progress bar
		bar.Increment()
	}

	// Stop progress bar
	bar.Stop()

	return &allResources, nil
}

// NukeAllResources nukes all GCP resources
// The context should contain a reporting.Collector for event-driven reporting.
func NukeAllResources(ctx context.Context, account *GcpProjectResources, configObj config.Config, bar *pterm.ProgressbarPrinter) {
	resourcesInRegion := account.Resources["global"]

	for _, gcpResource := range resourcesInRegion.Resources {
		nukeResource(ctx, gcpResource, configObj, bar)
	}
}

// nukeResource nukes a single GCP resource type
func nukeResource(parentCtx context.Context, gcpResource *GcpResource, configObj config.Config, bar *pterm.ProgressbarPrinter) {
	// Filter to only nukable resources
	var nukableIdentifiers []string
	for _, id := range (*gcpResource).ResourceIdentifiers() {
		if nukable, reason := (*gcpResource).IsNukable(id); !nukable {
			logging.Debugf("[Skipping] %s %s because %v", (*gcpResource).ResourceName(), id, reason)
			continue
		}
		nukableIdentifiers = append(nukableIdentifiers, id)
	}

	if len(nukableIdentifiers) == 0 {
		return
	}

	// Create context with timeout from resource config, preserving collector from parent
	ctx := parentCtx
	resourceConfig := (*gcpResource).GetAndSetResourceConfig(configObj)
	if resourceConfig.Timeout != "" {
		if duration, err := time.ParseDuration(resourceConfig.Timeout); err == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, duration)
			defer cancel()
		}
	}

	// Split API calls into batches
	logging.Debugf("Terminating %d %s in batches", len(nukableIdentifiers), (*gcpResource).ResourceName())
	batches := util.Split(nukableIdentifiers, (*gcpResource).MaxBatchSize())

	for i := 0; i < len(batches); i++ {
		batch := batches[i]
		bar.UpdateTitle(fmt.Sprintf("Nuking batch of %d %s resource(s)",
			len(batch), (*gcpResource).ResourceName()))
		if err := (*gcpResource).Nuke(ctx, batch); err != nil {
			if strings.Contains(err.Error(), "QUOTA_EXCEEDED") {
				logging.Debug(
					"Quota exceeded. Waiting 1 minute before making new requests",
				)
				time.Sleep(1 * time.Minute)
				continue
			}

			// Report to telemetry - aggregated metrics of failures per resources.
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: fmt.Sprintf("error:Nuke:%s", (*gcpResource).ResourceName()),
			}, map[string]interface{}{})
		}

		if i != len(batches)-1 {
			logging.Debug("Sleeping for 10 seconds before processing next batch...")
			time.Sleep(10 * time.Second)
		}

		// Update the spinner to show the current resource type being nuked
		bar.Add(len(batch))
	}
}

// getAllResourceTypes - Returns all GCP resource types that can be deleted
func getAllResourceTypes() []GcpResource {
	return []GcpResource{
		resources.NewGCSBuckets(),
	}
}

// ListResourceTypes - Returns list of resources which can be passed to --resource-type
func ListResourceTypes() []string {
	resourceTypes := []string{}
	for _, r := range getAllResourceTypes() {
		resourceTypes = append(resourceTypes, r.ResourceName())
	}
	return resourceTypes
}
