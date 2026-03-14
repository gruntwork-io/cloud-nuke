package gcp

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/gcp/resources"

	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/collections"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"github.com/hashicorp/go-multierror"
)

// IsNukeable checks whether a resource type should be nuked based on the
// requested resource types and exclude lists. An empty include list or the
// special value "all" means nuke everything, minus any excluded types.
func IsNukeable(resourceType string, resourceTypes []string, excludeResourceTypes []string) bool {
	if collections.ListContainsElement(excludeResourceTypes, resourceType) {
		return false
	}
	if len(resourceTypes) == 0 ||
		collections.ListContainsElement(resourceTypes, "all") ||
		collections.ListContainsElement(resourceTypes, resourceType) {
		return true
	}
	return false
}

// GetAllResources lists all GCP resources that can be deleted.
func GetAllResources(ctx context.Context, query *Query, configObj config.Config, collector *reporting.Collector) (*GcpProjectResources, error) {
	allResources := GcpProjectResources{
		Resources: map[string]GcpResources{},
	}

	ctx = context.WithValue(ctx, util.ExcludeFirstSeenTagKey, query.ExcludeFirstSeen)

	for _, region := range query.Regions {
		cfg := resources.GcpConfig{ProjectID: query.ProjectID, Region: region}
		regionResources := GetAndInitRegisteredResources(cfg, region)

		for _, res := range regionResources {
			resourceName := (*res).ResourceName()

			if !IsNukeable(resourceName, query.ResourceTypes, query.ExcludeResourceTypes) {
				continue
			}

			// Emit scan progress event
			collector.Emit(reporting.ScanProgress{
				ResourceType: resourceName,
				Region:       region,
			})

			// Get all resource identifiers
			identifiers, err := (*res).GetAndSetIdentifiers(ctx, configObj)
			if err != nil {
				if isServiceDisabledError(err) && !collections.ListContainsElement(query.ResourceTypes, resourceName) {
					logging.Debugf("Skipping %s: API is disabled in this project", resourceName)
					continue
				}
				logging.Debugf("Error getting identifiers for %s: %v", resourceName, err)
				collector.Emit(reporting.GeneralError{
					ResourceType: resourceName,
					Description:  fmt.Sprintf("Unable to retrieve %s", resourceName),
					Error:        err.Error(),
				})
				continue
			}

			// Only append if we have non-empty identifiers
			if len(identifiers) > 0 {
				logging.Infof("Found %d %s resources", len(identifiers), resourceName)
				allResources.Resources[region] = GcpResources{
					Resources: append(allResources.Resources[region].Resources, res),
				}

				// Emit ResourceFound events for each identifier
				for _, id := range identifiers {
					nukable, reason := true, ""
					if _, err := (*res).IsNukable(id); err != nil {
						nukable, reason = false, err.Error()
					}
					collector.Emit(reporting.ResourceFound{
						ResourceType: resourceName,
						Region:       region,
						Identifier:   id,
						Nukable:      nukable,
						Reason:       reason,
					})
				}
			}
		}
	}

	logging.Info("Done searching for GCP resources")
	logging.Infof("Found total of %d GCP resources", allResources.TotalResourceCount())

	return &allResources, nil
}

// NukeAllResources nukes all GCP resources across the given regions.
func NukeAllResources(ctx context.Context, account *GcpProjectResources, regions []string, collector *reporting.Collector) error {
	// Emit NukeStarted event (CLIRenderer will initialize progress bar)
	collector.Emit(reporting.NukeStarted{Total: account.TotalResourceCount()})

	var allErrors *multierror.Error

	for _, region := range regions {
		if err := nukeAllResourcesInRegion(ctx, account, region, collector); err != nil {
			allErrors = multierror.Append(allErrors, err)
		}
	}

	// Emit NukeComplete event (triggers final output in renderers)
	collector.Emit(reporting.NukeComplete{})

	return allErrors.ErrorOrNil()
}

// nukeAllResourcesInRegion nukes all resources in a single region.
func nukeAllResourcesInRegion(ctx context.Context, account *GcpProjectResources, region string, collector *reporting.Collector) error {
	var allErrors *multierror.Error

	resourcesInRegion := account.Resources[region]
	for _, gcpResource := range resourcesInRegion.Resources {
		if err := nukeResource(ctx, gcpResource, region, collector); err != nil {
			allErrors = multierror.Append(allErrors, err)
		}
	}

	return allErrors.ErrorOrNil()
}

// nukeResource nukes a single GCP resource type
func nukeResource(ctx context.Context, gcpResource *GcpResource, region string, collector *reporting.Collector) error {
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

	// Split API calls into batches
	logging.Debugf("Terminating %d %s in batches", len(nukableIdentifiers), (*gcpResource).ResourceName())
	batches := util.Split(nukableIdentifiers, (*gcpResource).MaxBatchSize())

	var allErrors *multierror.Error

	for i, batch := range batches {
		// Emit progress event (CLIRenderer updates its progress bar)
		collector.Emit(reporting.NukeProgress{
			ResourceType: (*gcpResource).ResourceName(),
			Region:       region,
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
				Region:       region,
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

			allErrors = multierror.Append(allErrors, fmt.Errorf("[%s] %s: %w", region, (*gcpResource).ResourceName(), err))

			// Report to telemetry - aggregated metrics of failures per resources.
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: fmt.Sprintf("error:Nuke:%s", (*gcpResource).ResourceName()),
			}, map[string]interface{}{
				"region": region,
			})
		}

		if i != len(batches)-1 {
			logging.Debug("Sleeping for 10 seconds before processing next batch...")
			time.Sleep(10 * time.Second)
		}
	}

	return allErrors.ErrorOrNil()
}

// ListResourceTypes returns a sorted list of resources which can be passed to --resource-type
func ListResourceTypes() []string {
	var resourceTypes []string
	for _, r := range GetAllRegisteredResources() {
		resourceTypes = append(resourceTypes, (*r).ResourceName())
	}
	sort.Strings(resourceTypes)
	return resourceTypes
}
