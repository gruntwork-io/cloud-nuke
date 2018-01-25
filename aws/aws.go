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

// Returns a list of all resource identifiers in an AWS region from given account resources
func getResourceIdenfiersForRegion(account *AwsAccountResources, resource string, region string) []string {
	for _, res := range account.Resources[resource] {
		if res.RegionName == region {
			return res.ResourceIdentifiers
		}
	}

	return make([]string, 0)
}

// GetAllResources - Lists all aws resources
func GetAllResources() (*AwsAccountResources, error) {
	account := AwsAccountResources{
		Resources: make(map[string][]AwsRegionResources),
	}

	for _, region := range getAllRegions() {
		session, err := session.NewSession(&awsgo.Config{
			Region: awsgo.String(region)},
		)

		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		instanceIds, err := getAllEc2Instances(session, region)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		regionResources := AwsRegionResources{
			RegionName:          region,
			ResourceIdentifiers: awsgo.StringValueSlice(instanceIds),
		}

		account.Resources["ec2"] = append(account.Resources["ec2"], regionResources)
	}

	return &account, nil
}

// NukeAllResources - Nukes all aws resources
func NukeAllResources(account *AwsAccountResources) error {
	for _, region := range getAllRegions() {
		session, err := session.NewSession(&awsgo.Config{
			Region: awsgo.String(region)},
		)

		if err != nil {
			return errors.WithStackTrace(err)
		}

		instanceIds := awsgo.StringSlice(getResourceIdenfiersForRegion(account, "ec2", region))
		if err := nukeAllEc2Instances(session, instanceIds); err != nil {
			return errors.WithStackTrace(err)
		}
	}

	return nil
}
