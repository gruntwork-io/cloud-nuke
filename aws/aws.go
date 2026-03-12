package aws

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/collections"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"github.com/hashicorp/go-multierror"
)

// GetAllResources - Lists all aws resources
func GetAllResources(c context.Context, query *Query, configObj config.Config, collector *reporting.Collector) (*AwsAccountResources, error) {
	configObj.AddExcludeAfterTime(query.ExcludeAfter)
	configObj.AddIncludeAfterTime(query.IncludeAfter)
	configObj.AddTimeout(query.Timeout)
	// Only override the config file value if the CLI flag is explicitly set to true
	if query.ListUnaliasedKMSKeys {
		configObj.KMSCustomerKeys.IncludeUnaliasedKeys = true
	}

	// Setting the DefaultOnly field
	// This function only sets the objects that have the `DefaultOnly` field, currently VPC, Subnet, and Security Group.
	configObj.AddEC2DefaultOnly(query.DefaultOnly)

	// This will protect dated resources by nuking them until the specified date has passed
	configObj.AddProtectUntilExpireFlag(query.ProtectUntilExpire)

	account := AwsAccountResources{
		Resources: make(map[string]AwsResources),
	}

	c = context.WithValue(c, util.ExcludeFirstSeenTagKey, query.ExcludeFirstSeen)
	scanCb := awsScanCallbacks(collector)

	for _, region := range query.Regions {
		cloudNukeSession, errSession := NewSession(region)
		if errSession != nil {
			return nil, errSession
		}

		accountId, err := util.GetCurrentAccountId(cloudNukeSession)
		if err == nil {
			telemetry.SetAccountId(accountId)
			c = context.WithValue(c, util.AccountIdKey, accountId)
		}

		awsResource := AwsResources{}
		registeredResources := GetAndInitRegisteredResources(cloudNukeSession, region)
		for _, res := range registeredResources {
			if !util.IsNukeable((*res).ResourceName(), query.ResourceTypes, nil) {
				continue
			}

			identifiers := resource.ScanResource(c, *res, region, configObj, scanCb)
			if len(identifiers) > 0 {
				awsResource.Resources = append(awsResource.Resources, res)
			}
		}

		if len(awsResource.Resources) > 0 {
			account.Resources[region] = awsResource
		}
	}

	logging.Info("Done searching for resources")
	logging.Infof("Found total of %d resources", account.TotalResourceCount())

	return &account, nil
}

// ListResourceTypes - Returns list of resources which can be passed to --resource-type
func ListResourceTypes() []string {
	resourceTypes := []string{}
	for _, resource := range GetAllRegisteredResources() {
		resourceTypes = append(resourceTypes, (*resource).ResourceName())
	}

	sort.Strings(resourceTypes)
	return resourceTypes
}

// IsValidResourceType - Checks if a resourceType is valid or not
func IsValidResourceType(resourceType string, allResourceTypes []string) bool {
	return collections.ListContainsElement(allResourceTypes, resourceType)
}

// IsNukeable checks whether a resource type should be nuked based on the
// requested resource types. This is a convenience wrapper around util.IsNukeable
// for backward compatibility.
func IsNukeable(resourceType string, resourceTypes []string) bool {
	return util.IsNukeable(resourceType, resourceTypes, nil)
}

func nukeAllResourcesInRegion(ctx context.Context, account *AwsAccountResources, region string, collector *reporting.Collector) error {
	var allErrors *multierror.Error
	resourcesInRegion := account.Resources[region]
	nukeCb := awsNukeCallbacks(collector)

	for _, awsResource := range resourcesInRegion.Resources {
		if err := resource.NukeInBatches(ctx, *awsResource, region, nukeCb); err != nil {
			allErrors = multierror.Append(allErrors, err)
		}
	}

	return allErrors.ErrorOrNil()
}

// NukeAllResources - Nukes all aws resources
func NukeAllResources(ctx context.Context, account *AwsAccountResources, regions []string, collector *reporting.Collector) error {
	// Emit NukeStarted event (CLIRenderer will initialize progress bar)
	collector.Emit(reporting.NukeStarted{Total: account.TotalResourceCount()})

	var allErrors *multierror.Error

	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Begin nuking resources",
	}, map[string]interface{}{})

	for _, region := range regions {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Creating session for region",
		}, map[string]interface{}{
			"region": region,
		})

		if err := nukeAllResourcesInRegion(ctx, account, region, collector); err != nil {
			allErrors = multierror.Append(allErrors, err)
		}
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Done Nuking Region",
		}, map[string]interface{}{
			"region":        region,
			"resourceCount": len(account.Resources[region].Resources),
		})
	}

	// Emit NukeComplete event (triggers final output in renderers)
	collector.Emit(reporting.NukeComplete{})

	return allErrors.ErrorOrNil()
}

// awsScanCallbacks creates scan callbacks wired to the reporting collector and telemetry.
func awsScanCallbacks(collector *reporting.Collector) resource.ScanCallbacks {
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
	}
}

// awsNukeCallbacks creates nuke callbacks wired to the reporting collector and telemetry.
func awsNukeCallbacks(collector *reporting.Collector) resource.NukeBatchCallbacks {
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
		IsRetryableError: util.IsThrottlingError,
	}
}
