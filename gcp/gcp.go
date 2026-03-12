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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/collections"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"github.com/hashicorp/go-multierror"
)

// IsNukeable checks whether a resource type should be nuked based on the
// requested resource types and exclude lists. This is a convenience wrapper
// around util.IsNukeable for backward compatibility.
func IsNukeable(resourceType string, resourceTypes []string, excludeResourceTypes []string) bool {
	return util.IsNukeable(resourceType, resourceTypes, excludeResourceTypes)
}

// GetAllResources lists all GCP resources that can be deleted.
func GetAllResources(ctx context.Context, query *Query, configObj config.Config, collector *reporting.Collector) (*GcpProjectResources, error) {
	allResources := GcpProjectResources{
		Resources: map[string]GcpResources{},
	}

	scanCb := gcpScanCallbacks(collector, query.ResourceTypes)

	for _, region := range query.Regions {
		cfg := resources.GcpConfig{ProjectID: query.ProjectID, Region: region}
		regionResources := GetAndInitRegisteredResources(cfg, region)

		for _, res := range regionResources {
			resourceName := (*res).ResourceName()

			if !IsNukeable(resourceName, query.ResourceTypes, query.ExcludeResourceTypes) {
				continue
			}

			identifiers := resource.ScanResource(ctx, *res, region, configObj, scanCb)
			if len(identifiers) > 0 {
				allResources.Resources[region] = GcpResources{
					Resources: append(allResources.Resources[region].Resources, res),
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
	nukeCb := gcpNukeCallbacks(collector)

	resourcesInRegion := account.Resources[region]
	for _, gcpResource := range resourcesInRegion.Resources {
		if err := resource.NukeInBatches(ctx, *gcpResource, region, nukeCb); err != nil {
			allErrors = multierror.Append(allErrors, err)
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

// gcpScanCallbacks creates scan callbacks wired to the reporting collector and telemetry.
// The requestedTypes parameter is used to determine if service-disabled errors should be skipped.
func gcpScanCallbacks(collector *reporting.Collector, requestedTypes []string) resource.ScanCallbacks {
	return resource.ScanCallbacks{
		OnScanProgress: func(resourceName, region string) {
			collector.Emit(reporting.ScanProgress{
				ResourceType: resourceName,
				Region:       region,
			})
		},
		OnScanError: func(resourceName, region string, err error) {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: fmt.Sprintf("error:GetIdentifiers:%s", resourceName),
			}, map[string]interface{}{
				"region": region,
			})
			collector.Emit(reporting.GeneralError{
				ResourceType: resourceName,
				Description:  fmt.Sprintf("Unable to retrieve %s", resourceName),
				Error:        err.Error(),
			})
		},
		OnScanComplete: func(resourceName string, count int, duration time.Duration) {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: fmt.Sprintf("Done getting %s identifiers", resourceName),
			}, map[string]interface{}{
				"recordCount": count,
				"actionTime":  duration.Seconds(),
			})
		},
		OnResourceFound: func(resourceName, region, id string, nukable bool, reason string) {
			collector.Emit(reporting.ResourceFound{
				ResourceType: resourceName,
				Region:       region,
				Identifier:   id,
				Nukable:      nukable,
				Reason:       reason,
			})
		},
		ShouldSkipError: func(resourceName string, err error) bool {
			// Skip service-disabled errors only when the resource wasn't explicitly requested
			return isServiceDisabledError(err) && !collections.ListContainsElement(requestedTypes, resourceName)
		},
	}
}

// gcpNukeCallbacks creates nuke callbacks wired to the reporting collector and telemetry.
func gcpNukeCallbacks(collector *reporting.Collector) resource.NukeBatchCallbacks {
	return resource.NukeBatchCallbacks{
		OnBatchStart: func(resourceName, region string, batchSize int) {
			collector.Emit(reporting.NukeProgress{
				ResourceType: resourceName,
				Region:       region,
				BatchSize:    batchSize,
			})
		},
		OnResult: func(resourceName, region string, result resource.NukeResult) {
			errStr := ""
			if result.Error != nil {
				errStr = result.Error.Error()
			}
			collector.Emit(reporting.ResourceDeleted{
				ResourceType: resourceName,
				Region:       region,
				Identifier:   result.Identifier,
				Success:      result.Error == nil,
				Error:        errStr,
			})
		},
		OnNukeError: func(resourceName, region string, err error) {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: fmt.Sprintf("error:Nuke:%s", resourceName),
			}, map[string]interface{}{
				"region": region,
			})
		},
		IsRetryableError: isQuotaExhaustedError,
	}
}
