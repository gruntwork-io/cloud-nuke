package aws

import (
	"fmt"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/collections"
)

// ExtractResourcesForPrinting is a convenience method that converts the nested structure of AwsAccountResources
// into a flat slice of resource identifiers, well-suited for printing line by line
func ExtractResourcesForPrinting(account *AwsAccountResources) []string {
	var resources []string

	if len(account.Resources) == 0 {
		logging.Logger.Infoln("No resources found!")
		return resources
	}

	resources = make([]string, 0)
	for region, resourcesInRegion := range account.Resources {
		for _, foundResources := range resourcesInRegion.Resources {
			for _, identifier := range foundResources.ResourceIdentifiers() {
				resources = append(resources, fmt.Sprintf("* %s %s %s\n", foundResources.ResourceName(), identifier, region))
			}
		}
	}

	return resources
}

func ensureValidResourceTypes(resourceTypes, excludeResourceTypes, allResourceTypes []string) ([]string, error) {

	invalidresourceTypes := []string{}
	for _, resourceType := range resourceTypes {
		if resourceType == "all" {
			continue
		}
		if !IsValidResourceType(resourceType, allResourceTypes) {
			invalidresourceTypes = append(invalidresourceTypes, resourceType)
		}
	}

	for _, resourceType := range excludeResourceTypes {
		if !IsValidResourceType(resourceType, allResourceTypes) {
			invalidresourceTypes = append(invalidresourceTypes, resourceType)
		}
	}

	if len(invalidresourceTypes) > 0 {
		return []string{}, InvalidResourceTypesSuppliedError{InvalidTypes: invalidresourceTypes}
	}

	return resourceTypes, nil
}

func HandleResourceTypeSelections(resourceTypes, excludeResourceTypes []string) ([]string, error) {
	if len(resourceTypes) > 0 && len(excludeResourceTypes) > 0 {
		return []string{}, ResourceTypeAndExcludeFlagsBothPassedError{}
	}
	// Ensure only selected resource types are being targeted
	allResourceTypes := ListResourceTypes()

	validResourceTypes, err := ensureValidResourceTypes(resourceTypes, excludeResourceTypes, allResourceTypes)
	if err != nil {
		return validResourceTypes, err
	}

	// Handle exclude resource types by going through the list of all types and only include those that are not
	// mentioned in the exclude list.
	if len(excludeResourceTypes) > 0 {
		for _, resourceType := range allResourceTypes {
			if !collections.ListContainsElement(excludeResourceTypes, resourceType) {
				resourceTypes = append(resourceTypes, resourceType)
			}
		}
	}

	return resourceTypes, nil
}

func InspectResources(q Query) (*AwsAccountResources, error) {

	account := &AwsAccountResources{
		Resources: make(map[string]AwsRegionResource),
	}

	resourceTypes, err := HandleResourceTypeSelections(q.ResourceTypes, q.ExcludeResourceTypes)
	if err != nil {
		return account, err
	}

	// Log which resource types will be inspected
	logging.Logger.Info("The following resource types will be scanned for (inspected):")
	if len(resourceTypes) > 0 {
		for _, resourceType := range resourceTypes {
			logging.Logger.Infof("- %s", resourceType)
		}
	} else {
		for _, resourceType := range ListResourceTypes() {
			logging.Logger.Infof("- %s", resourceType)
		}
	}

	if err != nil {
		return account, err
	}

	// We're currently short-circuiting the config file by passing an empty config here,
	// but the inspect functionalty does not currently support the config file
	return GetAllResources(q.Regions, q.ExcludeAfter, resourceTypes, config.Config{})
}
