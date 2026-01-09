package aws

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/pterm/pterm"

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

	spinner, _ := pterm.DefaultSpinner.WithRemoveWhenDone(true).Start()
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
			resourceName := (*resource).ResourceName()
			if !IsNukeable(resourceName, query.ResourceTypes) {
				continue
			}

			resourceConfig := (*resource).GetAndSetResourceConfig(configObj)
			if err := (*resource).PrepareContext(c, resourceConfig); err != nil {
				return nil, err
			}

			spinner.UpdateText(fmt.Sprintf("Searching %s resources in %s", resourceName, region))
			start := time.Now()
			identifiers, err := (*resource).GetAndSetIdentifiers(c, configObj)
			if err != nil {
				logging.Errorf("Unable to retrieve %s: %v", resourceName, err)
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: fmt.Sprintf("error:GetIdentifiers:%s", resourceName),
				}, map[string]interface{}{"region": region})
				collector.RecordError(resourceName, fmt.Sprintf("Unable to retrieve %s", resourceName), err)
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: fmt.Sprintf("Done getting %s identifiers", resourceName),
			}, map[string]interface{}{
				"recordCount": len(identifiers),
				"actionTime":  time.Since(start).Seconds(),
			})

			// Report found resources to collector (for inspect operations)
			for _, id := range identifiers {
				nukable, reason := (*resource).IsNukable(id)
				reasonStr := ""
				if reason != nil {
					reasonStr = reason.Error()
				}
				collector.RecordFound(resourceName, region, id, nukable, reasonStr)
			}

			if len(identifiers) > 0 {
				pterm.Info.Println(fmt.Sprintf("Found %d %s resources in %s", len(identifiers), resourceName, region))
				awsResource.Resources = append(awsResource.Resources, resource)
			}
		}

		if len(awsResource.Resources) > 0 {
			account.Resources[region] = awsResource
		}
	}

	logging.Info("Done searching for resources")
	logging.Infof("Found total of %d resources", account.TotalResourceCount())
	err := spinner.Stop()
	if err != nil {
		return nil, err
	}

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

func nukeAllResourcesInRegion(ctx context.Context, account *AwsAccountResources, region string, bar *pterm.ProgressbarPrinter, collector *reporting.Collector) {
	resourcesInRegion := account.Resources[region]

	for _, awsResource := range resourcesInRegion.Resources {
		resourceName := (*awsResource).ResourceName()
		length := len((*awsResource).ResourceIdentifiers())

		// Split api calls into batches
		logging.Debugf("Terminating %d awsResource in batches", length)
		batches := util.Split((*awsResource).ResourceIdentifiers(), (*awsResource).MaxBatchSize())

		for i := 0; i < len(batches); i++ {
			batch := batches[i]
			bar.UpdateTitle(fmt.Sprintf("Nuking batch of %d %s resource(s) in %s",
				len(batch), resourceName, region))

			results := (*awsResource).Nuke(ctx, batch)

			// Report results via collector
			hasErrors := false
			for _, result := range results {
				collector.RecordDeleted(resourceName, region, result.Identifier, result.Error)
				if result.Error != nil {
					hasErrors = true
					// Check for rate limiting
					if strings.Contains(result.Error.Error(), "RequestLimitExceeded") {
						logging.Debug(
							"Request limit reached. Waiting 1 minute before making new requests",
						)
						time.Sleep(1 * time.Minute)
					}
				}
			}

			// Report to telemetry on errors
			if hasErrors {
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: fmt.Sprintf("error:Nuke:%s", resourceName),
				}, map[string]interface{}{
					"region": region,
				})
			}

			if i != len(batches)-1 {
				logging.Debug("Sleeping for 10 seconds before processing next batch...")
				time.Sleep(10 * time.Second)
			}

			// Update the spinner to show the current resource type being nuked
			bar.Add(len(batch))
		}
	}
}

// NukeAllResources - Nukes all aws resources
func NukeAllResources(ctx context.Context, account *AwsAccountResources, regions []string, collector *reporting.Collector) error {
	// Set the progressbar width to the total number of nukeable resources found
	// across all regions
	progressBar, err := pterm.DefaultProgressbar.WithTotal(account.TotalResourceCount()).Start()
	if err != nil {
		return err
	}

	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Begin nuking resources",
	}, map[string]interface{}{})

	for _, region := range regions {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Creating session for region",
		}, map[string]interface{}{
			"region": region,
		})

		// We intentionally do not handle an error returned from this method, because we collect individual errors
		// on per-resource basis via the collector. In the run report displayed at the end of
		// a cloud-nuke run, we show exactly which resources deleted cleanly and which encountered errors
		nukeAllResourcesInRegion(ctx, account, region, progressBar, collector)
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Done Nuking Region",
		}, map[string]interface{}{
			"region":        region,
			"resourceCount": len(account.Resources[region].Resources),
		})
	}

	_, err = progressBar.Stop()
	if err != nil {
		return err
	}

	return nil
}
