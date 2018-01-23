package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
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
func GetAllResources() ([]string, error) {
	var resources []string
	for _, region := range getAllRegions() {
		session, err := session.NewSession(&awsgo.Config{
			Region: awsgo.String(region)},
		)

		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		instances, err := getAllEc2Instances(session, region)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		resources = append(resources, instances...)
	}

	return resources, nil
}

// NukeAllResources - Nukes all aws resources
func NukeAllResources() error {
	for _, region := range getAllRegions() {
		session, err := session.NewSession(&awsgo.Config{
			Region: awsgo.String(region)},
		)

		if err != nil {
			return errors.WithStackTrace(err)
		}

		err = nukeAllEc2Instances(session)
		if err != nil {
			return errors.WithStackTrace(err)
		}
	}

	return nil
}
