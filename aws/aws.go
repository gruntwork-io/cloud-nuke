package aws

import (
	"math/rand"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/collections"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// GetAllRegions - Returns a list of all AWS regions
func GetAllRegions() []string {
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
	allRegions := GetAllRegions()
	rand.Seed(time.Now().UnixNano())
	randIndex := rand.Intn(len(allRegions))
	return allRegions[randIndex]
}

func split(identifiers []string, limit int) [][]string {
	var chunk []string
	chunks := make([][]string, 0, len(identifiers)/limit+1)
	for len(identifiers) >= limit {
		chunk, identifiers = identifiers[:limit], identifiers[limit:]
		chunks = append(chunks, chunk)
	}
	if len(identifiers) > 0 {
		chunks = append(chunks, identifiers[:len(identifiers)])
	}

	return chunks
}

// GetAllResources - Lists all aws resources
func GetAllResources(regions []string, excludedRegions []string, excludeAfter time.Time) (*AwsAccountResources, error) {
	account := AwsAccountResources{
		Resources: make(map[string]AwsRegionResource),
	}

	for _, region := range regions {
		// Ignore all cli excluded regions
		if collections.ListContainsElement(excludedRegions, region) {
			logging.Logger.Infoln("Skipping region: " + region)
			continue
		}

		session, err := session.NewSession(&awsgo.Config{
			Region: awsgo.String(region)},
		)

		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		resourcesInRegion := AwsRegionResource{}

		// The order in which resources are nuked is important
		// because of dependencies between resources

		// ASG Names
		groupNames, err := getAllAutoScalingGroups(session, region, excludeAfter)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		asGroups := ASGroups{
			GroupNames: awsgo.StringValueSlice(groupNames),
		}

		resourcesInRegion.Resources = append(resourcesInRegion.Resources, asGroups)
		// End ASG Names

		// LoadBalancer Names
		elbNames, err := getAllElbInstances(session, region, excludeAfter)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		loadBalancers := LoadBalancers{
			Names: awsgo.StringValueSlice(elbNames),
		}

		resourcesInRegion.Resources = append(resourcesInRegion.Resources, loadBalancers)
		// End LoadBalancer Names

		// LoadBalancerV2 Arns
		elbv2Arns, err := getAllElbv2Instances(session, region, excludeAfter)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		loadBalancersV2 := LoadBalancersV2{
			Arns: awsgo.StringValueSlice(elbv2Arns),
		}

		resourcesInRegion.Resources = append(resourcesInRegion.Resources, loadBalancersV2)
		// End LoadBalancerV2 Arns

		// EC2 Instances
		instanceIds, err := getAllEc2Instances(session, region, excludeAfter)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		ec2Instances := EC2Instances{
			InstanceIds: awsgo.StringValueSlice(instanceIds),
		}

		resourcesInRegion.Resources = append(resourcesInRegion.Resources, ec2Instances)
		// End EC2 Instances

		// EBS Volumes
		volumeIds, err := getAllEbsVolumes(session, region, excludeAfter)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		ebsVolumes := EBSVolumes{
			VolumeIds: awsgo.StringValueSlice(volumeIds),
		}

		resourcesInRegion.Resources = append(resourcesInRegion.Resources, ebsVolumes)
		// End EBS Volumes

		// AMIs
		imageIds, err := getAllAMIs(session, region, excludeAfter)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		amis := AMIs{
			ImageIds: awsgo.StringValueSlice(imageIds),
		}

		resourcesInRegion.Resources = append(resourcesInRegion.Resources, amis)
		// End AMIs

		// Snapshots
		snapshotIds, err := getAllSnapshots(session, region, excludeAfter)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		snapshots := Snapshots{
			SnapshotIds: awsgo.StringValueSlice(snapshotIds),
		}

		resourcesInRegion.Resources = append(resourcesInRegion.Resources, snapshots)
		// End Snapshots

		account.Resources[region] = resourcesInRegion
	}

	return &account, nil
}

// NukeAllResources - Nukes all aws resources
func NukeAllResources(account *AwsAccountResources, regions []string) error {
	for _, region := range regions {
		session, err := session.NewSession(&awsgo.Config{
			Region: awsgo.String(region)},
		)

		if err != nil {
			return errors.WithStackTrace(err)
		}

		resourcesInRegion := account.Resources[region]
		for _, resources := range resourcesInRegion.Resources {
			length := len(resources.ResourceIdentifiers())

			// Split api calls into batches
			logging.Logger.Infof("Terminating %d resources in batches", length)
			batches := split(resources.ResourceIdentifiers(), resources.MaxBatchSize())

			for _, batch := range batches {
				if err := resources.Nuke(session, batch); err != nil {
					return errors.WithStackTrace(err)
				}

				time.Sleep(10 * time.Second)
			}
		}
	}

	return nil
}
