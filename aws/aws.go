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
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/collections"
)

// GetAllResources - Lists all aws resources
func GetAllResources(c context.Context, query *Query, configObj config.Config) (*AwsAccountResources, error) {
	configObj.AddExcludeAfterTime(query.ExcludeAfter)
	configObj.AddIncludeAfterTime(query.IncludeAfter)
	configObj.AddTimeout(query.Timeout)

	configObj.KMSCustomerKeys.IncludeUnaliasedKeys = query.ListUnaliasedKMSKeys
	account := AwsAccountResources{
		Resources: make(map[string]AwsResources),
	}

	spinner, _ := pterm.DefaultSpinner.WithRemoveWhenDone(true).Start()
	for _, region := range query.Regions {
		cloudNukeSession := NewSession(region)
		accountId, err := util.GetCurrentAccountId(cloudNukeSession)
		if err == nil {
			telemetry.SetAccountId(accountId)
			c = context.WithValue(c, util.AccountIdKey, accountId)
		}

		awsResource := AwsResources{}
		registeredResources := GetAndInitRegisteredResources(cloudNukeSession, region)
		for _, resource := range registeredResources {
			if IsNukeable((*resource).ResourceName(), query.ResourceTypes) {

				// PrepareContext sets up the resource context for execution, utilizing the context 'c' and the resource individual configuration.
				// This function should be called after configuring the timeout to ensure proper execution context.
				resourceConfig := (*resource).GetAndSetResourceConfig(configObj)
				(*resource).PrepareContext(c, resourceConfig)

				spinner.UpdateText(
					fmt.Sprintf("Searching %s resources in %s", (*resource).ResourceName(), region))
				start := time.Now()
				identifiers, err := (*resource).GetAndSetIdentifiers(c, configObj)
				if err != nil {
					logging.Errorf("Unable to retrieve %v, %v", (*resource).ResourceName(), err)
					ge := report.GeneralError{
						Error:        err,
						Description:  fmt.Sprintf("Unable to retrieve %s", (*resource).ResourceName()),
						ResourceType: (*resource).ResourceName(),
					}
					report.RecordError(ge)
				}

				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: fmt.Sprintf("Done getting %s identifiers", (*resource).ResourceName()),
				}, map[string]interface{}{
					"recordCount": len(identifiers),
					"actionTime":  time.Since(start).Seconds(),
				})

				// Only append if we have non-empty identifiers
				if len(identifiers) > 0 {
					pterm.Info.Println(fmt.Sprintf("Found %d %s resources in %s", len(identifiers), (*resource).ResourceName(), region))
					awsResource.Resources = append(awsResource.Resources, resource)
				}
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

func nukeAllResourcesInRegion(account *AwsAccountResources, region string, bar *pterm.ProgressbarPrinter) {
	resourcesInRegion := account.Resources[region]

	for _, awsResource := range resourcesInRegion.Resources {
		length := len((*awsResource).ResourceIdentifiers())

		// Split api calls into batches
		logging.Debugf("Terminating %d awsResource in batches", length)
		batches := util.Split((*awsResource).ResourceIdentifiers(), (*awsResource).MaxBatchSize())

		for i := 0; i < len(batches); i++ {
			batch := batches[i]
			bar.UpdateTitle(fmt.Sprintf("Nuking batch of %d %s resource(s) in %s",
				len(batch), (*awsResource).ResourceName(), region))
			if err := (*awsResource).Nuke(batch); err != nil {
				// TODO: Figure out actual error type
				if strings.Contains(err.Error(), "RequestLimitExceeded") {
					logging.Debug(
						"Request limit reached. Waiting 1 minute before making new requests",
					)
					time.Sleep(1 * time.Minute)
					continue
				}

				// We're only interested in acting on Rate limit errors - no other error should prevent further processing
				// of the current job.Since we handle each individual resource deletion error within its own resource-specific code,
				// we can safely discard this error
				_ = err
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
func NukeAllResources(account *AwsAccountResources, regions []string) error {
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
		// on per-resource basis via the report package's Record method. In the run report displayed at the end of
		// a cloud-nuke run, we show exactly which resources deleted cleanly and which encountered errors
		nukeAllResourcesInRegion(account, region, progressBar)
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
