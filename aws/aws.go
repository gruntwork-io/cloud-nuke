package aws

import (
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/util"
	"sort"
	"strings"
	"time"

	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/progressbar"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/collections"
)

// GetAllResources - Lists all aws resources
func GetAllResources(
	targetRegions []string,
	excludeAfter time.Time,
	resourceTypes []string,
	configObj config.Config,
	allowDeleteUnaliasedKeys bool) (*AwsAccountResources, error) {

	configObj.AddExcludeAfterTime(&excludeAfter)
	configObj.KMSCustomerKeys.DeleteUnaliasedKeys = allowDeleteUnaliasedKeys
	account := AwsAccountResources{
		Resources: make(map[string]AwsRegionResource),
	}

	for _, region := range targetRegions {
		logging.Logger.Infoln("Listing resources in region: ", region)
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
				logging.Logger.Infoln("Listing resources of type: ", (*resource).ResourceName())
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
					awsResource.Resources = append(awsResource.Resources, resource)
				}
			}
		}

		if len(awsResource.Resources) > 0 {
			account.Resources[region] = awsResource
		}
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

func nukeAllResourcesInRegion(account *AwsAccountResources, region string, session *session.Session) {
	resourcesInRegion := account.Resources[region]

	for _, resources := range resourcesInRegion.Resources {
		length := len((*resources).ResourceIdentifiers())

		// Split api calls into batches
		logging.Logger.Debugf("Terminating %d resources in batches", length)
		batches := util.Split((*resources).ResourceIdentifiers(), (*resources).MaxBatchSize())

		for i := 0; i < len(batches); i++ {
			batch := batches[i]
			if err := (*resources).Nuke(batch); err != nil {
				// TODO: Figure out actual error type
				if strings.Contains(err.Error(), "RequestLimitExceeded") {
					logging.Logger.Debug(
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
				logging.Logger.Debug("Sleeping for 10 seconds before processing next batch...")
				time.Sleep(10 * time.Second)
			}
		}
	}
}

// NukeAllResources - Nukes all aws resources
func NukeAllResources(account *AwsAccountResources, regions []string) error {
	// Set the progressbar width to the total number of nukeable resources found
	// across all regions
	progressbar.StartProgressBarWithLength(account.TotalResourceCount())

	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Begin nuking resources",
	}, map[string]interface{}{})

	defaultRegion := regions[0]
	for _, region := range regions {
		// region that will be used to create a session
		sessionRegion := region

		// As there is no actual region named global we have to pick a valid one just to create the session
		if region == GlobalRegion {
			sessionRegion = defaultRegion
		}

		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Creating session for region",
		}, map[string]interface{}{
			"region": region,
		})
		session := NewSession(sessionRegion)
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Nuking Region",
		}, map[string]interface{}{
			"region":        region,
			"resourceCount": len(account.Resources[region].Resources),
		})

		// We intentionally do not handle an error returned from this method, because we collect individual errors
		// on per-resource basis via the report package's Record method. In the run report displayed at the end of
		// a cloud-nuke run, we show exactly which resources deleted cleanly and which encountered errors
		nukeAllResourcesInRegion(account, region, session)
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Done Nuking Region",
		}, map[string]interface{}{
			"region":        region,
			"resourceCount": len(account.Resources[region].Resources),
		})
	}

	return nil
}
