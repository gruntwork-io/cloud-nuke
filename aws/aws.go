package aws

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/hashicorp/go-multierror"

	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/reporting"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/collections"
)

// GetAllResources - Lists all aws resources
func GetAllResources(c context.Context, query *Query, configObj config.Config, collector *reporting.Collector) (*AwsAccountResources, error) {
	configObj.AddExcludeAfterTime(query.ExcludeAfter)
	configObj.AddIncludeAfterTime(query.IncludeAfter)
	configObj.AddIncludeTags(query.IncludeTags)
	configObj.AddTimeout(query.Timeout)
	// Only override the config file value if the CLI flag is explicitly set to true
	if query.ListUnaliasedKMSKeys {
		configObj.KMSCustomerKeys.IncludeUnaliasedKeys = true
	}

	// Setting the DefaultOnly field
	// This function only sets the objects that have the `DefaultOnly` field, currently VPC, Subnet, and Security Group.
	configObj.AddEC2DefaultOnly(query.DefaultOnly)

	account := AwsAccountResources{
		Resources: make(map[string]AwsResources),
	}

	c = context.WithValue(c, util.ExcludeFirstSeenTagKey, query.ExcludeFirstSeen)
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
		for _, resource := range registeredResources {
			if IsNukeable((*resource).ResourceName(), query.ResourceTypes) {

				(*resource).GetAndSetResourceConfig(configObj)

				// Emit scan progress event
				collector.Emit(reporting.ScanProgress{
					ResourceType: (*resource).ResourceName(),
					Region:       region,
				})

				start := time.Now()
				identifiers, err := (*resource).GetAndSetIdentifiers(c, configObj)
				if err != nil {
					logging.Errorf("Unable to retrieve %v, %v", (*resource).ResourceName(), err)

					// Reporting resource-level failures encountered during the GetIdentifiers phase
					telemetry.TrackEvent(commonTelemetry.EventContext{
						EventName: fmt.Sprintf("error:GetIdentifiers:%s", (*resource).ResourceName()),
					}, map[string]interface{}{
						"region": region,
					})

					collector.Emit(reporting.GeneralError{
						ResourceType: (*resource).ResourceName(),
						Description:  fmt.Sprintf("Unable to retrieve %s", (*resource).ResourceName()),
						Error:        err.Error(),
					})
				}

				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: fmt.Sprintf("Done getting %s identifiers", (*resource).ResourceName()),
				}, map[string]interface{}{
					"recordCount": len(identifiers),
					"actionTime":  time.Since(start).Seconds(),
				})

				// Only append if we have non-empty identifiers
				if len(identifiers) > 0 {
					logging.Infof("Found %d %s resources in %s", len(identifiers), (*resource).ResourceName(), region)
					awsResource.Resources = append(awsResource.Resources, resource)

					// Emit ResourceFound events for each identifier
					for _, id := range identifiers {
						nukable, reason := true, ""
						if _, err := (*resource).IsNukable(id); err != nil {
							nukable, reason = false, err.Error()
						}
						collector.Emit(reporting.ResourceFound{
							ResourceType: (*resource).ResourceName(),
							Region:       region,
							Identifier:   id,
							Nukable:      nukable,
							Reason:       reason,
						})
					}
				}
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

// IsNukeable - Checks if we should nuke a resource or not
func IsNukeable(resourceType string, resourceTypes []string) bool {
	if len(resourceTypes) == 0 ||
		collections.ListContainsElement(resourceTypes, "all") ||
		collections.ListContainsElement(resourceTypes, resourceType) {
		return true
	}
	return false
}

func nukeAllResourcesInRegion(ctx context.Context, account *AwsAccountResources, region string, collector *reporting.Collector) error {
	var allErrors *multierror.Error
	resourcesInRegion := account.Resources[region]

	for _, awsResource := range resourcesInRegion.Resources {
		length := len((*awsResource).ResourceIdentifiers())

		// Split api calls into batches
		logging.Debugf("Terminating %d awsResource in batches", length)
		batches := util.Split((*awsResource).ResourceIdentifiers(), (*awsResource).MaxBatchSize())

		for i, batch := range batches {
			// Emit progress event (CLIRenderer updates its progress bar)
			collector.Emit(reporting.NukeProgress{
				ResourceType: (*awsResource).ResourceName(),
				Region:       region,
				BatchSize:    len(batch),
			})

			results, err := (*awsResource).Nuke(ctx, batch)

			// Emit ResourceDeleted for each result
			for _, result := range results {
				errStr := ""
				if result.Error != nil {
					errStr = result.Error.Error()
				}
				collector.Emit(reporting.ResourceDeleted{
					ResourceType: (*awsResource).ResourceName(),
					Region:       region,
					Identifier:   result.Identifier,
					Success:      result.Error == nil,
					Warning:      result.Error != nil && util.IsWarningError(result.Error),
					Error:        errStr,
				})
			}

			if err != nil {
				// Handle rate limiting
				if util.IsThrottlingError(err) {
					logging.Debug(
						"Request limit reached. Waiting 1 minute before making new requests",
					)
					time.Sleep(1 * time.Minute)
					continue
				}

				allErrors = multierror.Append(allErrors, fmt.Errorf("[%s] %s: %w", region, (*awsResource).ResourceName(), err))

				// Report to telemetry - aggregated metrics of failures per resources.
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: fmt.Sprintf("error:Nuke:%s", (*awsResource).ResourceName()),
				}, map[string]interface{}{
					"region": region,
				})
			}

			if i != len(batches)-1 {
				logging.Debug("Sleeping for 10 seconds before processing next batch...")
				time.Sleep(10 * time.Second)
			}
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
