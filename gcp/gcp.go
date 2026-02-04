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
)

// GetAllResources lists all GCP resources that can be deleted.
func GetAllResources(projectID string, configObj config.Config, excludeAfter time.Time, includeAfter time.Time, collector *reporting.Collector) (*GcpProjectResources, error) {
	allResources := GcpProjectResources{
		Resources: map[string]GcpResources{},
	}

	// Get all resource types to delete
	resourceTypes := getAllResourceTypes()

	// For each resource type
	for _, resourceType := range resourceTypes {
		// Emit scan progress event
		collector.Emit(reporting.ScanProgress{
			ResourceType: resourceType.ResourceName(),
			Region:       "global",
		})

		// Initialize the resource
		resourceType.Init(projectID)

		// Get all resource identifiers
		identifiers, err := resourceType.GetAndSetIdentifiers(context.Background(), configObj)
		if err != nil {
			logging.Debugf("Error getting identifiers for %s: %v", resourceType.ResourceName(), err)
			collector.Emit(reporting.GeneralError{
				ResourceType: resourceType.ResourceName(),
				Description:  fmt.Sprintf("Unable to retrieve %s", resourceType.ResourceName()),
				Error:        err.Error(),
			})
			continue
		}

		// Only append if we have non-empty identifiers
		if len(identifiers) > 0 {
			logging.Infof("Found %d %s resources", len(identifiers), resourceType.ResourceName())
			allResources.Resources["global"] = GcpResources{
				Resources: append(allResources.Resources["global"].Resources, &resourceType),
			}

			// Emit ResourceFound events for each identifier
			for _, id := range identifiers {
				nukable, reason := true, ""
				if _, err := resourceType.IsNukable(id); err != nil {
					nukable, reason = false, err.Error()
				}
				collector.Emit(reporting.ResourceFound{
					ResourceType: resourceType.ResourceName(),
					Region:       "global",
					Identifier:   id,
					Nukable:      nukable,
					Reason:       reason,
				})
			}
		}
	}

	logging.Info("Done searching for GCP resources")
	logging.Infof("Found total of %d GCP resources", allResources.TotalResourceCount())

	return &allResources, nil
}

// NukeAllResources nukes all GCP resources
func NukeAllResources(account *GcpProjectResources, configObj config.Config, collector *reporting.Collector) {
	// Emit NukeStarted event (CLIRenderer will initialize progress bar)
	collector.Emit(reporting.NukeStarted{Total: account.TotalResourceCount()})

	resourcesInRegion := account.Resources["global"]

	for _, gcpResource := range resourcesInRegion.Resources {
		nukeResource(gcpResource, configObj, collector)
	}

	// Emit NukeComplete event (triggers final output in renderers)
	collector.Emit(reporting.NukeComplete{})
}

// nukeResource nukes a single GCP resource type
func nukeResource(gcpResource *GcpResource, configObj config.Config, collector *reporting.Collector) {
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

	// Create context with timeout from resource config
	ctx := context.Background()
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

	for i, batch := range batches {
		// Emit progress event (CLIRenderer updates its progress bar)
		collector.Emit(reporting.NukeProgress{
			ResourceType: (*gcpResource).ResourceName(),
			Region:       "global",
			BatchSize:    len(batch),
		})

		results, err := (*gcpResource).Nuke(ctx, batch)

		// Emit ResourceDeleted for each result
		for _, result := range results {
			errStr := ""
			if result.Error != nil {
				errStr = result.Error.Error()
			}
			collector.Emit(reporting.ResourceDeleted{
				ResourceType: (*gcpResource).ResourceName(),
				Region:       "global",
				Identifier:   result.Identifier,
				Success:      result.Error == nil,
				Error:        errStr,
			})
		}

		if err != nil {
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
