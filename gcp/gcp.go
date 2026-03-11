package gcp

import (
	"context"
	"fmt"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/gcp/resources"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"github.com/hashicorp/go-multierror"
)

// GetAllResources lists all GCP resources that can be deleted.
func GetAllResources(ctx context.Context, projectID string, configObj config.Config, excludeAfter time.Time, includeAfter time.Time, collector *reporting.Collector) (*GcpProjectResources, error) {
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
		identifiers, err := resourceType.GetAndSetIdentifiers(ctx, configObj)
		if err != nil {
			if isServiceDisabledError(err) {
				logging.Debugf("Skipping %s: API is disabled in this project", resourceType.ResourceName())
				continue
			}
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
func NukeAllResources(ctx context.Context, account *GcpProjectResources, configObj config.Config, collector *reporting.Collector) error {
	// Emit NukeStarted event (CLIRenderer will initialize progress bar)
	collector.Emit(reporting.NukeStarted{Total: account.TotalResourceCount()})

	var allErrors *multierror.Error

	resourcesInRegion := account.Resources["global"]

	for _, gcpResource := range resourcesInRegion.Resources {
		if err := nukeResource(ctx, gcpResource, configObj, collector); err != nil {
			allErrors = multierror.Append(allErrors, err)
		}
	}

	// Emit NukeComplete event (triggers final output in renderers)
	collector.Emit(reporting.NukeComplete{})

	return allErrors.ErrorOrNil()
}

// nukeResource nukes a single GCP resource type
func nukeResource(ctx context.Context, gcpResource *GcpResource, configObj config.Config, collector *reporting.Collector) error {
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
		return nil
	}

	// Apply timeout from resource config on top of the caller-provided context
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

	var allErrors *multierror.Error

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
			if isQuotaExhaustedError(err) {
				logging.Debug(
					"Quota exceeded. Waiting 1 minute before making new requests",
				)
				time.Sleep(1 * time.Minute)
				continue
			}

			allErrors = multierror.Append(allErrors, fmt.Errorf("[global] %s: %w", (*gcpResource).ResourceName(), err))

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

	return allErrors.ErrorOrNil()
}

// getAllResourceTypes - Returns all GCP resource types that can be deleted
func getAllResourceTypes() []GcpResource {
	return []GcpResource{
		resources.NewGCSBuckets(),
		resources.NewCloudFunctions(),
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
