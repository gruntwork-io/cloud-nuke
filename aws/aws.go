package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
)

// Returns a list of all AWS regions
func getAllRegions() []string {
	resolver := endpoints.DefaultResolver()
	partitions := resolver.(endpoints.EnumPartitions).Partitions()

	var regions []string
	for _, p := range partitions {
		for id := range p.Regions() {
			regions = append(regions, id)
		}
	}

	return regions
}

// GetAllResources - Lists all aws resources
func GetAllResources() []string {
	var resources []string
	for _, region := range getAllRegions() {
		session, _ := session.NewSession(&awsgo.Config{
			Region: awsgo.String(region)},
		)

		instances, err := getAllEc2Instances(session, region)
		if err == nil {
			resources = append(resources, instances...)
		}
	}

	return resources
}

// NukeAllResources - Nukes all aws resources
func NukeAllResources() {
	for _, region := range getAllRegions() {
		session, _ := session.NewSession(&awsgo.Config{
			Region: awsgo.String(region)},
		)

		nukeAllEc2Instances(session)
	}
}
