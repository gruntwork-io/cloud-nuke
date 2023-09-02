package aws

import (
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/pterm/pterm"
	"sort"
	"strings"
	"time"

	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/collections"
)

// GetAllResources - Lists all aws resources
func GetAllResources(
	targetRegions []string,
	excludeAfter *time.Time,
	resourceTypes []string,
	configObj config.Config,
	allowDeleteUnaliasedKeys bool) (*AwsAccountResources, error) {

	configObj.AddExcludeAfterTime(excludeAfter)
	configObj.KMSCustomerKeys.IncludeUnaliasedKeys = allowDeleteUnaliasedKeys
	account := AwsAccountResources{
		Resources: make(map[string]AwsRegionResource),
	}

	spinner, _ := pterm.DefaultSpinner.WithRemoveWhenDone(true).Start()
	for _, region := range targetRegions {
		cloudNukeSession := NewSession(region)
		stsService := sts.New(cloudNukeSession)
		resp, err := stsService.GetCallerIdentity(&sts.GetCallerIdentityInput{})
		if err == nil {
			telemetry.SetAccountId(*resp.Account)
		}

		awsResource := AwsRegionResource{}
		registeredResources := GetAndInitRegisteredResources(cloudNukeSession, region)
		for _, resource := range registeredResources {
			if IsNukeable((*resource).ResourceName(), resourceTypes) {
				spinner.UpdateText(
					fmt.Sprintf("Searching %s resources in %s", (*resource).ResourceName(), region))
				start := time.Now()
				identifiers, err := (*resource).GetAndSetIdentifiers(configObj)
				if err != nil {
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

	pterm.Info.Println("Done searching for resources")
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

func nukeAllResourcesInRegion(account *AwsAccountResources, region string) error {
	spinner, err := pterm.DefaultSpinner.WithRemoveWhenDone(true).Start()
	if err != nil {
		return err
	}

	resourcesInRegion := account.Resources[region]
	for _, resources := range resourcesInRegion.Resources {
		spinner.UpdateText(fmt.Sprintf(
			"Nuking %s resources in %s", (*resources).ResourceName(), region))
		length := len((*resources).ResourceIdentifiers())

		// Split api calls into batches
		pterm.Debug.Println(fmt.Sprintf("Terminating %d resources in batches", length))
		batches := util.Split((*resources).ResourceIdentifiers(), (*resources).MaxBatchSize())

		for i := 0; i < len(batches); i++ {
			batch := batches[i]
			err := (*resources).Nuke(batch)
			if err != nil && strings.Contains(err.Error(), "RequestLimitExceeded") {
				pterm.Debug.Println(
					"Request limit reached. Waiting 1 minute before making new requests")
				time.Sleep(1 * time.Minute)
				continue
			}

			if err != nil {
				pterm.Error.Println(fmt.Sprintf("Encountered errors while nuking %s resources in %s [batch %d]",
					(*resources).ResourceName(), region, i))
			} else {
				pterm.Info.Println(fmt.Sprintf("Nuked %d %s resources in %s [batch %d]",
					len(batch), (*resources).ResourceName(), region, i))
			}

			// Note: Sleep for 10 seconds between batches to avoid throttling
			if i != len(batches)-1 {
				pterm.Debug.Println("Sleeping for 10 seconds before processing next batch...")
				time.Sleep(10 * time.Second)
			}
		}
	}

	return nil
}

// NukeAllResources - Nukes all aws resources
func NukeAllResources(account *AwsAccountResources, regions []string) error {
	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Begin nuking resources",
	}, map[string]interface{}{})

	for _, region := range regions {
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Creating session for region",
		}, map[string]interface{}{
			"region": region,
		})
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Nuking Region",
		}, map[string]interface{}{
			"region":        region,
			"resourceCount": len(account.Resources[region].Resources),
		})

		// We intentionally do not handle an error returned from this method, because we collect individual errors
		// on per-resource basis via the report package's Record method. In the run report displayed at the end of
		// a cloud-nuke run, we show exactly which resources deleted cleanly and which encountered errors
		err := nukeAllResourcesInRegion(account, region)
		if err != nil {
			return err
		}

		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Done Nuking Region",
		}, map[string]interface{}{
			"region":        region,
			"resourceCount": len(account.Resources[region].Resources),
		})
	}

	return nil
}
