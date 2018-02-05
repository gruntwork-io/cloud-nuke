package aws

import (
	"math/rand"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/collections"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Returns a list of all AWS regions
func getAllRegions() []string {
	// chinese and government regions are not accessible with regular accounts
	reservedRegions := []string{
		"cn-north-1", "cn-northwest-1", "us-gov-west-1",
	}

	resolver := endpoints.DefaultResolver()
	partitions := resolver.(endpoints.EnumPartitions).Partitions()

	var regions []string
	for _, p := range partitions {
		for id := range p.Regions() {
			if !collections.ListContainsElement(reservedRegions, id) {
				regions = append(regions, id)
			}
		}
	}

	return regions
}

func getRandomRegion() string {
	allRegions := getAllRegions()
	rand.Seed(time.Now().UnixNano())
	randIndex := rand.Intn(len(allRegions))
	return allRegions[randIndex]
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

		// LoadBalancerV2 Arns
		elbv2Arns, err := getAllElbv2Instances(session, region)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		loadBalancersV2 := LoadBalancersV2{
			Arns: awsgo.StringValueSlice(elbv2Arns),
		}

		resourcesInRegion.Resources = append(resourcesInRegion.Resources, loadBalancersV2)
		// End LoadBalancerV2 Arns

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

		// EBS Volumes
		volumeIds, err := getAllEbsVolumes(session, region)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		ebsVolumes := EBSVolumes{
			VolumeIds: awsgo.StringValueSlice(volumeIds),
		}

		resourcesInRegion.Resources = append(resourcesInRegion.Resources, ebsVolumes)
		// End EBS Volumes

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
