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
func GetAllResources() (*AwsAccountResources, error) {
	account := AwsAccountResources{
		Resources: make(map[string]AwsRegionResource),
	}

	for _, region := range getAllRegions() {
		session, err := session.NewSession(&awsgo.Config{
			Region: awsgo.String(region)},
		)

		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		resourcesInRegion := AwsRegionResource{}

		// ASG Names
		groupNames, err := getAllAutoScalingGroups(session, region)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		asGroups := ASGroups{
			GroupNames: awsgo.StringValueSlice(groupNames),
		}

		resourcesInRegion.Resources = append(resourcesInRegion.Resources, asGroups)
		// End ASG Names

		// LoadBalancer Names
		elbNames, err := getAllElbInstances(session, region)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		loadBalancers := LoadBalancers{
			Names: awsgo.StringValueSlice(elbNames),
		}

		resourcesInRegion.Resources = append(resourcesInRegion.Resources, loadBalancers)
		// End LoadBalancer Names

		// EC2 Instances
		instanceIds, err := getAllEc2Instances(session, region)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		ec2Instances := EC2Instances{
			InstanceIds: awsgo.StringValueSlice(instanceIds),
		}

		resourcesInRegion.Resources = append(resourcesInRegion.Resources, ec2Instances)
		// End EC2 Instances

		account.Resources[region] = resourcesInRegion
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

		resourcesInRegion := account.Resources[region]
		for _, resources := range resourcesInRegion.Resources {
			if err := resources.Nuke(session); err != nil {
				return errors.WithStackTrace(err)
			}
		}
	}

	return nil
}
