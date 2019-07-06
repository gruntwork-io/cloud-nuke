package aws

import (
	"math/rand"
	"sort"
	"strings"
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
		"cn-north-1", "cn-northwest-1", "us-gov-west-1", "us-gov-east-1",
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
	if limit < 0 {
		limit = -1 * limit
	} else if limit == 0 {
		return [][]string{identifiers}
	}

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
func GetAllResources(regions []string, excludedRegions []string, excludeAfter time.Time, resourceTypes []string) (*AwsAccountResources, error) {
	account := AwsAccountResources{
		Resources: make(map[string]AwsRegionResource),
	}

	for _, region := range regions {
		// Ignore all cli excluded regions
		if collections.ListContainsElement(excludedRegions, region) {
			logging.Logger.Infoln("Skipping region: " + region)
			continue
		}
		logging.Logger.Infoln("Checking region: " + region)

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
		asGroups := ASGroups{}
		if IsNukeable(asGroups.ResourceName(), resourceTypes) {
			groupNames, err := getAllAutoScalingGroups(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			asGroups.GroupNames = awsgo.StringValueSlice(groupNames)
			resourcesInRegion.Resources = append(resourcesInRegion.Resources, asGroups)
		}
		// End ASG Names

		// Launch Configuration Names
		configs := LaunchConfigs{}
		if IsNukeable(configs.ResourceName(), resourceTypes) {
			configNames, err := getAllLaunchConfigurations(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			configs.LaunchConfigurationNames = awsgo.StringValueSlice(configNames)
			resourcesInRegion.Resources = append(resourcesInRegion.Resources, configs)
		}
		// End Launch Configuration Names

		// LoadBalancer Names
		loadBalancers := LoadBalancers{}
		if IsNukeable(loadBalancers.ResourceName(), resourceTypes) {
			elbNames, err := getAllElbInstances(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			loadBalancers.Names = awsgo.StringValueSlice(elbNames)
			resourcesInRegion.Resources = append(resourcesInRegion.Resources, loadBalancers)
		}
		// End LoadBalancer Names

		// LoadBalancerV2 Arns
		loadBalancersV2 := LoadBalancersV2{}
		if IsNukeable(loadBalancersV2.ResourceName(), resourceTypes) {
			elbv2Arns, err := getAllElbv2Instances(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			loadBalancersV2.Arns = awsgo.StringValueSlice(elbv2Arns)
			resourcesInRegion.Resources = append(resourcesInRegion.Resources, loadBalancersV2)
		}
		// End LoadBalancerV2 Arns

		// EC2 Instances
		ec2Instances := EC2Instances{}
		if IsNukeable(ec2Instances.ResourceName(), resourceTypes) {
			instanceIds, err := getAllEc2Instances(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			ec2Instances.InstanceIds = awsgo.StringValueSlice(instanceIds)
			resourcesInRegion.Resources = append(resourcesInRegion.Resources, ec2Instances)
		}
		// End EC2 Instances

		// EBS Volumes
		ebsVolumes := EBSVolumes{}
		if IsNukeable(ebsVolumes.ResourceName(), resourceTypes) {
			volumeIds, err := getAllEbsVolumes(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			ebsVolumes.VolumeIds = awsgo.StringValueSlice(volumeIds)
			resourcesInRegion.Resources = append(resourcesInRegion.Resources, ebsVolumes)
		}
		// End EBS Volumes

		// EIP Addresses
		eipAddresses := EIPAddresses{}
		if IsNukeable(eipAddresses.ResourceName(), resourceTypes) {
			allocationIds, err := getAllEIPAddresses(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			eipAddresses.AllocationIds = awsgo.StringValueSlice(allocationIds)
			resourcesInRegion.Resources = append(resourcesInRegion.Resources, eipAddresses)
		}
		// End EIP Addresses

		// AMIs
		amis := AMIs{}
		if IsNukeable(amis.ResourceName(), resourceTypes) {
			imageIds, err := getAllAMIs(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			amis.ImageIds = awsgo.StringValueSlice(imageIds)
			resourcesInRegion.Resources = append(resourcesInRegion.Resources, amis)
		}
		// End AMIs

		// Snapshots
		snapshots := Snapshots{}
		if IsNukeable(snapshots.ResourceName(), resourceTypes) {
			snapshotIds, err := getAllSnapshots(session, region, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			snapshots.SnapshotIds = awsgo.StringValueSlice(snapshotIds)
			resourcesInRegion.Resources = append(resourcesInRegion.Resources, snapshots)
		}
		// End Snapshots

		// ECS resources
		ecsServices := ECSServices{}
		if IsNukeable(ecsServices.ResourceName(), resourceTypes) {
			clusterArns, err := getAllEcsClusters(session)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			serviceArns, serviceClusterMap, err := getAllEcsServices(session, clusterArns, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			ecsServices.Services = awsgo.StringValueSlice(serviceArns)
			ecsServices.ServiceClusterMap = serviceClusterMap
			resourcesInRegion.Resources = append(resourcesInRegion.Resources, ecsServices)
		}
		// End ECS resources

		// EKS resources
		eksClusters := EKSClusters{}
		if IsNukeable(eksClusters.ResourceName(), resourceTypes) {
			if eksSupportedRegion(region) {
				eksClusterNames, err := getAllEksClusters(session, excludeAfter)
				if err != nil {
					return nil, errors.WithStackTrace(err)
				}

				eksClusters.Clusters = awsgo.StringValueSlice(eksClusterNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, eksClusters)
			}
		}
		// End EKS resources

		if len(resourcesInRegion.Resources) > 0 {
			account.Resources[region] = resourcesInRegion
		}
	}

	return &account, nil
}

// ListResourceTypes - Returns list of resources which can be passed to --resource-type
func ListResourceTypes() []string {
	resourceTypes := []string{
		ASGroups{}.ResourceName(),
		LaunchConfigs{}.ResourceName(),
		LoadBalancers{}.ResourceName(),
		LoadBalancersV2{}.ResourceName(),
		EC2Instances{}.ResourceName(),
		EBSVolumes{}.ResourceName(),
		EIPAddresses{}.ResourceName(),
		AMIs{}.ResourceName(),
		Snapshots{}.ResourceName(),
		ECSServices{}.ResourceName(),
		EKSClusters{}.ResourceName(),
	}
	sort.Strings(resourceTypes)
	return resourceTypes
}

// IsValidResourceType - Checks if a resourceType is valid or not
func IsValidResourceType(resourceType string, allResourceTypes []string) bool {
	return collections.ListContainsElement(allResourceTypes, resourceType)
}

// IsNukeable - Checks if we should nuke a resource or not
func IsNukeable(resourceType string, resourceTypes []string) bool {
	if len(resourceTypes) == 0 ||
		collections.ListContainsElement(resourceTypes, "all") ||
		collections.ListContainsElement(resourceTypes, resourceType) {
		return true
	}
	return false
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

			for i := 0; i < len(batches); i++ {
				batch := batches[i]
				if err := resources.Nuke(session, batch); err != nil {
					// TODO: Figure out actual error type
					if strings.Contains(err.Error(), "RequestLimitExceeded") {
						logging.Logger.Info("Request limit reached. Waiting 1 minute before making new requests")
						time.Sleep(1 * time.Minute)
						continue
					}

					return errors.WithStackTrace(err)
				}

				if i != len(batches)-1 {
					logging.Logger.Info("Sleeping for 10 seconds before processing next batch...")
					time.Sleep(10 * time.Second)
				}
			}
		}
	}

	return nil
}
