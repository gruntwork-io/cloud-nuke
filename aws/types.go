package aws

import (
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/config"
	"strings"
	"time"
)

const AwsResourceExclusionTagKey = "cloud-nuke-excluded"

type AwsAccountResources struct {
	Resources map[string]AwsRegionResource
}

func (a *AwsAccountResources) GetRegion(region string) AwsRegionResource {
	if val, ok := a.Resources[region]; ok {
		return val
	}
	return AwsRegionResource{}
}

// TotalResourceCount returns the number of resources found, that are eligible for nuking, across all AWS regions targeted
// In other words, if you have 3 nukeable resources in us-east-1 and 4 nukeable resources in ap-southeast-1, this function
// would return 7
func (a *AwsAccountResources) TotalResourceCount() int {
	total := 0
	for _, regionResource := range a.Resources {
		for _, resource := range regionResource.Resources {
			total += len(resource.ResourceIdentifiers())
		}
	}
	return total
}

// MapResourceNameToIdentifiers converts a slice of Resources to a map of resource types to their found identifiers
// For example: ["ec2"] = ["i-0b22a22eec53b9321", "i-0e22a22yec53b9456"]
func (arr AwsRegionResource) MapResourceNameToIdentifiers() map[string][]string {
	// Initialize map of resource name to identifier, e.g., ["ec2"] = "i-0b22a22eec53b9321"
	m := make(map[string][]string)
	for _, resource := range arr.Resources {
		if len(resource.ResourceIdentifiers()) > 0 {
			for _, id := range resource.ResourceIdentifiers() {
				m[resource.ResourceName()] = append(m[resource.ResourceName()], id)
			}
		}
	}
	return m
}

// CountOfResourceType is a convenience method that returns the number of the supplied resource type found in the AwsRegionResource
func (arr AwsRegionResource) CountOfResourceType(resourceType string) int {
	idMap := arr.MapResourceNameToIdentifiers()
	resourceType = strings.ToLower(resourceType)
	if val, ok := idMap[resourceType]; ok {
		return len(val)
	}
	return 0
}

// ResourceTypePresent is a convenience method that returns true, if the given resource is found in the AwsRegionResource, or false if it is not
func (arr AwsRegionResource) ResourceTypePresent(resourceType string) bool {
	return arr.CountOfResourceType(resourceType) > 0
}

// IdentifiersForResourceType is a convenience method that returns the list of resource identifiers for a given resource type, if available
func (arr AwsRegionResource) IdentifiersForResourceType(resourceType string) []string {
	idMap := arr.MapResourceNameToIdentifiers()
	resourceType = strings.ToLower(resourceType)
	if val, ok := idMap[resourceType]; ok {
		return val
	}
	return []string{}
}

type AwsResources interface {
	ResourceName() string
	ResourceIdentifiers() []string
	MaxBatchSize() int
	Nuke(identifiers []string) error
	GetAndSetIdentifiers(configObj config.Config) ([]string, error)
}

type AwsRegionResource struct {
	Resources []AwsResources
}

// Query is a struct that represents the desired parameters for scanning resources within a given account
type Query struct {
	Regions              []string
	ExcludeRegions       []string
	ResourceTypes        []string
	ExcludeResourceTypes []string
	ExcludeAfter         time.Time
	ListUnaliasedKMSKeys bool
}

// NewQuery configures and returns a Query struct that can be passed into the InspectResources method
func NewQuery(regions, excludeRegions, resourceTypes, excludeResourceTypes []string, excludeAfter time.Time, listUnaliasedKMSKeys bool) (*Query, error) {
	q := &Query{
		Regions:              regions,
		ExcludeRegions:       excludeRegions,
		ResourceTypes:        resourceTypes,
		ExcludeResourceTypes: excludeResourceTypes,
		ExcludeAfter:         excludeAfter,
		ListUnaliasedKMSKeys: listUnaliasedKMSKeys,
	}

	validationErr := q.Validate()

	if validationErr != nil {
		return q, validationErr
	}

	return q, nil
}

// Validate ensures the configured values for a Query are valid, returning an error if there are
// any invalid params, or nil if the Query is valid
func (q *Query) Validate() error {
	resourceTypes, err := HandleResourceTypeSelections(q.ResourceTypes, q.ExcludeResourceTypes)
	if err != nil {
		return err
	}

	q.ResourceTypes = resourceTypes

	regions, err := GetEnabledRegions()
	if err != nil {
		return CouldNotDetermineEnabledRegionsError{Underlying: err}
	}

	// global is a fake region, used to represent global resources
	regions = append(regions, GlobalRegion)

	targetRegions, err := GetTargetRegions(regions, q.Regions, q.ExcludeRegions)
	if err != nil {
		return CouldNotSelectRegionError{Underlying: err}
	}

	q.Regions = targetRegions

	return nil
}

// custom errors

type InvalidResourceTypesSuppliedError struct {
	InvalidTypes []string
}

func (err InvalidResourceTypesSuppliedError) Error() string {
	return fmt.Sprintf("Invalid resourceTypes %s specified: %s", err.InvalidTypes, "Try --list-resource-types to get a list of valid resource types.")
}

type ResourceTypeAndExcludeFlagsBothPassedError struct{}

func (err ResourceTypeAndExcludeFlagsBothPassedError) Error() string {
	return "You can not specify both --resource-type and --exclude-resource-type"
}

type InvalidTimeStringPassedError struct {
	Entry      string
	Underlying error
}

func (err InvalidTimeStringPassedError) Error() string {
	return fmt.Sprintf("Could not parse %s as a valid time duration. Underlying error: %s", err.Entry, err.Underlying)
}

type QueryCreationError struct {
	Underlying error
}

func (err QueryCreationError) Error() string {
	return fmt.Sprintf("Error forming a cloud-nuke Query with supplied parameters. Original error: %v", err.Underlying)
}

type ResourceInspectionError struct {
	Underlying error
}

func (err ResourceInspectionError) Error() string {
	return fmt.Sprintf("Error encountered when querying for account resources. Original error: %v", err.Underlying)
}

type CouldNotSelectRegionError struct {
	Underlying error
}

func (err CouldNotSelectRegionError) Error() string {
	return fmt.Sprintf("Unable to determine target region set. Please double check your combination of target and excluded regions. Original error: %v", err.Underlying)
}

type CouldNotDetermineEnabledRegionsError struct {
	Underlying error
}

func (err CouldNotDetermineEnabledRegionsError) Error() string {
	return fmt.Sprintf("Unable to determine enabled regions in target account. Original error: %v", err.Underlying)
}
