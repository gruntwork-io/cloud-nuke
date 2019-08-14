package aws

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/collections"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// OptInNotRequiredRegions contains all regions that are enabled by default on new AWS accounts
// Beginning in Spring 2019, AWS requires new regions to be explicitly enabled
// See https://aws.amazon.com/blogs/security/setting-permissions-to-enable-accounts-for-upcoming-aws-regions/
var OptInNotRequiredRegions = [...]string{
	"eu-north-1",
	"ap-south-1",
	"eu-west-3",
	"eu-west-2",
	"eu-west-1",
	"ap-northeast-2",
	"ap-northeast-1",
	"sa-east-1",
	"ca-central-1",
	"ap-southeast-1",
	"ap-southeast-2",
	"eu-central-1",
	"us-east-1",
	"us-east-2",
	"us-west-1",
	"us-west-2",
}

func newSession(region string) *session.Session {
	return session.Must(
		session.NewSessionWithOptions(
			session.Options{
				SharedConfigState: session.SharedConfigEnable,
				Config: awsgo.Config{
					Region: awsgo.String(region),
				},
			},
		),
	)
}

// Try a describe regions command with the most likely enabled regions
func retryDescribeRegions() (*ec2.DescribeRegionsOutput, error) {
	for i := 0; i < len(OptInNotRequiredRegions); i++ {
		region := OptInNotRequiredRegions[rand.Intn(len(OptInNotRequiredRegions))]
		svc := ec2.New(newSession(region))
		regions, err := svc.DescribeRegions(&ec2.DescribeRegionsInput{})
		if err != nil {
			continue
		}
		return regions, nil
	}
	return nil, errors.WithStackTrace(fmt.Errorf("could not find any enabled regions"))
}

// Get all regions that are enabled (DescribeRegions excludes those not enabled by default)
func GetEnabledRegions() ([]string, error) {
	var regionNames []string

	// We don't want to depend on a default region being set, so instead we
	// will choose a region from the list of regions that are enabled by default
	// and use that to enumerate all enabled regions.
	// Corner case: user has intentionally disabled one or more regions that are
	// enabled by default. If that region is chosen, API calls will fail.
	// Therefore we retry until one of the regions works.
	regions, err := retryDescribeRegions()
	if err != nil {
		return nil, err
	}

	for _, region := range regions.Regions {
		regionNames = append(regionNames, awsgo.StringValue(region.RegionName))
	}

	return regionNames, nil
}

func getRandomRegion() string {
	allRegions, _ := GetEnabledRegions()
	rand.Seed(time.Now().UnixNano())
	randIndex := rand.Intn(len(allRegions))
	logging.Logger.Infof("Random region chosen: %s", allRegions[randIndex])
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

		// Launch Configuration Names
		configNames, err := getAllLaunchConfigurations(session, region, excludeAfter)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		configs := LaunchConfigs{
			LaunchConfigurationNames: awsgo.StringValueSlice(configNames),
		}

		resourcesInRegion.Resources = append(resourcesInRegion.Resources, configs)
		// End Launch Configuration Names

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

		// EIP Addresses
		allocationIds, err := getAllEIPAddresses(session, region, excludeAfter)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		eipAddresses := EIPAddresses{
			AllocationIds: awsgo.StringValueSlice(allocationIds),
		}

		resourcesInRegion.Resources = append(resourcesInRegion.Resources, eipAddresses)
		// End EIP Addresses

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

		// ECS resources
		clusterArns, err := getAllEcsClusters(session)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		serviceArns, serviceClusterMap, err := getAllEcsServices(session, clusterArns, excludeAfter)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		ecsServices := ECSServices{
			Services:          awsgo.StringValueSlice(serviceArns),
			ServiceClusterMap: serviceClusterMap,
		}
		resourcesInRegion.Resources = append(resourcesInRegion.Resources, ecsServices)
		// End ECS resources

		// EKS resources
		if eksSupportedRegion(region) {
			eksClusterNames, err := getAllEksClusters(session, excludeAfter)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			eksClusters := EKSClusters{
				Clusters: awsgo.StringValueSlice(eksClusterNames),
			}
			resourcesInRegion.Resources = append(resourcesInRegion.Resources, eksClusters)
		}
		// End EKS resources

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
