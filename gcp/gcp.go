package gcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/gcp/resources"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/collections"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"github.com/pterm/pterm"
)

// GetAllResources - Lists all GCP resources that can be deleted.
// If targetResourceTypes is empty, all resource types will be retrieved.
func GetAllResources(projectID string, configObj config.Config, excludeAfter time.Time, includeAfter time.Time, targetResourceTypes []string) (*GcpProjectResources, error) {
	allResources := GcpProjectResources{
		Resources: map[string]GcpResources{},
	}

	// Get all resource types to delete
	allResourceTypes := getAllResourceTypes()

	// Filter resource types if specified
	resourceTypes := allResourceTypes
	if len(targetResourceTypes) > 0 {
		resourceTypes = filterResourceTypes(allResourceTypes, targetResourceTypes)
	}

	// Create a progress bar
	bar, _ := pterm.DefaultProgressbar.WithTotal(len(resourceTypes)).WithTitle("Retrieving GCP resources").Start()

	// For each resource type
	for _, resourceType := range resourceTypes {
		// Update progress bar
		bar.UpdateTitle(fmt.Sprintf("Retrieving GCP %s", resourceType.ResourceName()))

		// Initialize the resource
		resourceType.Init(projectID)

		// Get the resource config
		resourceConfig := resourceType.GetAndSetResourceConfig(configObj)

		// Prepare context for the resource
		if err := resourceType.PrepareContext(context.Background(), resourceConfig); err != nil {
			logging.Debugf("Error preparing context for %s: %v", resourceType.ResourceName(), err)
			continue
		}

		// Get all resource identifiers
		if _, err := resourceType.GetAndSetIdentifiers(context.Background(), configObj); err != nil {
			logging.Debugf("Error getting identifiers for %s: %v", resourceType.ResourceName(), err)
			continue
		}

		// Add the resource to the map
		allResources.Resources["global"] = GcpResources{
			Resources: append(allResources.Resources["global"].Resources, &resourceType),
		}

		// Increment progress bar
		bar.Increment()
	}

	// Stop progress bar
	bar.Stop()

	return &allResources, nil
}

// NukeAllResources - Nukes all GCP resources
func NukeAllResources(account *GcpProjectResources, bar *pterm.ProgressbarPrinter) {
	resourcesInRegion := account.Resources["global"]

	for _, gcpResource := range resourcesInRegion.Resources {
		length := len((*gcpResource).ResourceIdentifiers())

		// Split API calls into batches
		logging.Debugf("Terminating %d gcpResource in batches", length)
		batches := splitIntoBatches((*gcpResource).ResourceIdentifiers(), (*gcpResource).MaxBatchSize())

		for i := 0; i < len(batches); i++ {
			batch := batches[i]
			bar.UpdateTitle(fmt.Sprintf("Nuking batch of %d %s resource(s)",
				len(batch), (*gcpResource).ResourceName()))
			if err := (*gcpResource).Nuke(batch); err != nil {
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

			// Update the spinner to show the current resource type being nuked
			bar.Add(len(batch))
		}
	}
}

// getAllResourceTypes - Returns all GCP resource types that can be deleted
func getAllResourceTypes() []GcpResource {
	return []GcpResource{
		&resources.GCSBuckets{},
	}
}

// ListResourceTypes - Returns list of resources which can be passed to --resource-type
func ListResourceTypes() []string {
	resourceTypes := []string{}
	for _, resource := range getAllResourceTypes() {
		resourceTypes = append(resourceTypes, resource.ResourceName())
	}
	return resourceTypes
}

// HandleResourceTypeSelections is a wrapper around the resource package function for GCP resource types.
// It filters any excluded or invalid types from target resourceTypes then returns the filtered slice.
func HandleResourceTypeSelections(
	includeResourceTypes, excludeResourceTypes []string,
) ([]string, error) {
	return resource.HandleResourceTypeSelections(includeResourceTypes, excludeResourceTypes, ListResourceTypes())
}

// filterResourceTypes filters the GCP resource types based on the target resource type names
func filterResourceTypes(allResources []GcpResource, targetNames []string) []GcpResource {
	filtered := []GcpResource{}
	for _, resource := range allResources {
		if collections.ListContainsElement(targetNames, resource.ResourceName()) {
			filtered = append(filtered, resource)
		}
	}
	return filtered
}

// splitIntoBatches - Splits a slice into batches
func splitIntoBatches(slice []string, batchSize int) [][]string {
	var batches [][]string
	for i := 0; i < len(slice); i += batchSize {
		end := i + batchSize
		if end > len(slice) {
			end = len(slice)
		}
		batches = append(batches, slice[i:end])
	}
	return batches
}
