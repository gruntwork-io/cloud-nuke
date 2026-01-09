package aws

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// AwsResource is an interface that represents a single AWS resource.
// Resources are unaware of reporting - they return results and the orchestration layer handles reporting.
type AwsResource interface {
	Init(cfg aws.Config)
	ResourceName() string
	ResourceIdentifiers() []string
	MaxBatchSize() int
	Nuke(ctx context.Context, identifiers []string) []resource.NukeResult
	GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error)
	IsNukable(string) (bool, error)

	PrepareContext(context.Context, config.ResourceType) error
	GetAndSetResourceConfig(config.Config) config.ResourceType
}

// AwsResources is a struct to hold multiple instances of AwsResource.
type AwsResources struct {
	Resources []*AwsResource
}

// AwsAccountResources is a struct that represents the resources found in a single AWS account
type AwsAccountResources struct {
	Resources map[string]AwsResources
}

func (a *AwsAccountResources) GetRegion(region string) AwsResources {
	if val, ok := a.Resources[region]; ok {
		return val
	}
	return AwsResources{}
}

// TotalResourceCount returns the number of resources found, that are eligible for nuking, across all AWS regions targeted
// In other words, if you have 3 nukeable resources in us-east-1 and 4 nukeable resources in ap-southeast-1, this function
// would return 7
func (a *AwsAccountResources) TotalResourceCount() int {
	total := 0
	for _, regionResource := range a.Resources {
		for _, resource := range regionResource.Resources {
			total += len((*resource).ResourceIdentifiers())
		}
	}
	return total
}

// MapResourceTypeToIdentifiers converts a slice of Resources to a map of resource types to their found identifiers
// For example: ["ec2"] = ["i-0b22a22eec53b9321", "i-0e22a22yec53b9456"]
func (arr *AwsResources) MapResourceTypeToIdentifiers() map[string][]string {
	// Initialize map of resource name to identifier, e.g., ["ec2"] = "i-0b22a22eec53b9321"
	m := make(map[string][]string)
	for _, resource := range arr.Resources {
		if len((*resource).ResourceIdentifiers()) > 0 {
			for _, id := range (*resource).ResourceIdentifiers() {
				m[(*resource).ResourceName()] = append(m[(*resource).ResourceName()], id)
			}
		}
	}
	return m
}

// CountOfResourceType is a convenience method that returns the number of the supplied resource type found
// in the AwsResources
func (arr *AwsResources) CountOfResourceType(resourceType string) int {
	idMap := arr.MapResourceTypeToIdentifiers()
	resourceType = strings.ToLower(resourceType)
	if val, ok := idMap[resourceType]; ok {
		return len(val)
	}
	return 0
}

// ResourceTypePresent is a convenience method that returns true, if the given resource is found in the AwsResources,
// or false if it is not
func (arr *AwsResources) ResourceTypePresent(resourceType string) bool {
	return arr.CountOfResourceType(resourceType) > 0
}

// IdentifiersForResourceType is a convenience method that returns the list of resource identifiers for a given
// resource type, if available
func (arr *AwsResources) IdentifiersForResourceType(resourceType string) []string {
	idMap := arr.MapResourceTypeToIdentifiers()
	resourceType = strings.ToLower(resourceType)
	if val, ok := idMap[resourceType]; ok {
		return val
	}
	return []string{}
}
