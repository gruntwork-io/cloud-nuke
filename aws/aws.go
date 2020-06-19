package aws

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/collections"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/sirupsen/logrus"
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

// CloudNukeAWSParams holds params for creating a new AWS session
type CloudNukeAWSParams struct {
	Region     string
	AWSSession *session.Session
}

// NewCloudNukeAWSParams - creates CloudNukeAWSParams type
func NewCloudNukeAWSParams(region string) (CloudNukeAWSParams, error) {
	var awsParams CloudNukeAWSParams
	if region == "" {
		var err error
		region, err = getRandomRegion()
		if err != nil {
			return awsParams, err
		}
		awsParams.Region = region
		logging.Logger.Debugf("Creating session in region - %s", region)
	} else {
		awsParams.Region = region
	}
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	awsParams.AWSSession = session
	return awsParams, err
}

// SetEnvLogLevel - sets log level from environment
func SetEnvLogLevel() error {
	logLevel := os.Getenv("LOG_LEVEL")
	if len(logLevel) > 0 {
		parsedLogLevel, err := logrus.ParseLevel(logLevel)
		if err != nil {
			return err
		}
		logging.Logger.Level = parsedLogLevel
	}
	return nil
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

// GetEnabledRegions - Get all regions that are enabled (DescribeRegions excludes those not enabled by default)
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

func getRandomRegion() (string, error) {
	allRegions, err := GetEnabledRegions()
	if err != nil {
		return "", errors.WithStackTrace(err)
	}
	rand.Seed(time.Now().UnixNano())
	randIndex := rand.Intn(len(allRegions))
	logging.Logger.Infof("Random region chosen: %s", allRegions[randIndex])
	return allRegions[randIndex], nil
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

// GetTargetRegions - Used enabled, selected and excluded regions to create a
// final list of valid regions
func GetTargetRegions(enabledRegions []string, selectedRegions []string, excludedRegions []string) ([]string, error) {
	if len(enabledRegions) == 0 {
		return nil, fmt.Errorf("Cannot have empty enabled regions")
	}

	// neither selectedRegions nor excludedRegions => select enabledRegions
	if len(selectedRegions) == 0 && len(excludedRegions) == 0 {
		return enabledRegions, nil
	}

	if len(selectedRegions) > 0 && len(excludedRegions) > 0 {
		return nil, fmt.Errorf("Cannot specify both selected and excluded regions")
	}

	var invalidRegions []string

	// Validate selectedRegions
	for _, selectedRegion := range selectedRegions {
		if !collections.ListContainsElement(enabledRegions, selectedRegion) {
			invalidRegions = append(invalidRegions, selectedRegion)
		}
	}
	if len(invalidRegions) > 0 {
		return nil, fmt.Errorf("Invalid values for region: [%s]", invalidRegions)
	}

	if len(selectedRegions) > 0 {
		return selectedRegions, nil
	}

	// Validate excludedRegions
	for _, excludedRegion := range excludedRegions {
		if !collections.ListContainsElement(enabledRegions, excludedRegion) {
			invalidRegions = append(invalidRegions, excludedRegion)
		}
	}
	if len(invalidRegions) > 0 {
		return nil, fmt.Errorf("Invalid values for exclude-region: [%s]", invalidRegions)
	}

	// Filter out excludedRegions from enabledRegions
	var targetRegions []string
	if len(excludedRegions) > 0 {
		for _, region := range enabledRegions {
			if !collections.ListContainsElement(excludedRegions, region) {
				targetRegions = append(targetRegions, region)
			}
		}
	}
	if len(targetRegions) == 0 {
		return nil, fmt.Errorf("Cannot exclude all regions: %s", excludedRegions)
	}
	return targetRegions, nil
}

// GetAllResourcesArgs encapsulates arguments for GetAllResources
type GetAllResourcesArgs struct {
	TargetRegions          []string                    // target regions to nuke
	ExcludeAfter           time.Time                   // exclude resources created after this time
	NukeResourceTypes      []string                    // type of resources to nuke
	IgnoreErrResourceTypes []string                    // ignore errors encountered while nuking resources of this type
	RegionSessionMap       map[string]*session.Session // use pre-created per region sessions if passed in
	ConfigObj              config.Config               // config object
}

// GetAllResources - Lists all aws resources
func GetAllResources(args GetAllResourcesArgs) (*AwsAccountResources, error) {
	targetRegions := args.TargetRegions
	excludeAfter := args.ExcludeAfter
	nukeResourceTypes := args.NukeResourceTypes
	ignoreErrResourceTypes := args.IgnoreErrResourceTypes
	regionSessionMap := args.RegionSessionMap

	account := AwsAccountResources{
		Resources: make(map[string]AwsRegionResource),
	}

	count := 1
	totalRegions := len(targetRegions)
	var resourcesCache = map[string]map[string][]*string{}

	for _, region := range targetRegions {
		logging.Logger.Infof("Checking region [%d/%d]: %s", count, totalRegions, region)

		sess, ok := regionSessionMap[region]
		if !ok {
			var err error
			sess, err = session.NewSession(&awsgo.Config{
				Region: awsgo.String(region)},
			)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
		}

		resourcesInRegion := AwsRegionResource{}

		// The order in which resources are nuked is important
		// because of dependencies between resources

		// ASG Names
		asGroups := ASGroups{}
		if IsNukeable(asGroups.ResourceName(), nukeResourceTypes) {
			groupNames, err := getAllAutoScalingGroups(sess, region, excludeAfter)
			if err != nil {
				if !IsErrIgnorable(asGroups.ResourceName(), ignoreErrResourceTypes) {
					return nil, errors.WithStackTrace(err)
				}
				logging.Logger.Warnf("Ignoring get resources error - %s - %s", asGroups.ResourceName(), err)
			} else {
				if len(groupNames) > 0 {
					asGroups.GroupNames = awsgo.StringValueSlice(groupNames)
					resourcesInRegion.Resources = append(resourcesInRegion.Resources, asGroups)
				}
			}
		}
		// End ASG Names

		// Launch Configuration Names
		configs := LaunchConfigs{}
		if IsNukeable(configs.ResourceName(), nukeResourceTypes) {
			configNames, err := getAllLaunchConfigurations(sess, region, excludeAfter)
			if err != nil {
				if !IsErrIgnorable(configs.ResourceName(), ignoreErrResourceTypes) {
					return nil, errors.WithStackTrace(err)
				}
				logging.Logger.Warnf("Ignoring get resources error - %s - %s", configs.ResourceName(), err)
			} else {
				if len(configNames) > 0 {
					configs.LaunchConfigurationNames = awsgo.StringValueSlice(configNames)
					resourcesInRegion.Resources = append(resourcesInRegion.Resources, configs)
				}
			}
		}
		// End Launch Configuration Names

		// LoadBalancer Names
		loadBalancers := LoadBalancers{}
		if IsNukeable(loadBalancers.ResourceName(), nukeResourceTypes) {
			elbNames, err := getAllElbInstances(sess, region, excludeAfter)
			if err != nil {
				if !IsErrIgnorable(loadBalancers.ResourceName(), ignoreErrResourceTypes) {
					return nil, errors.WithStackTrace(err)
				}
				logging.Logger.Warnf("Ignoring get resources error - %s - %s", loadBalancers.ResourceName(), err)
			} else {
				if len(elbNames) > 0 {
					loadBalancers.Names = awsgo.StringValueSlice(elbNames)
					resourcesInRegion.Resources = append(resourcesInRegion.Resources, loadBalancers)
				}
			}
		}
		// End LoadBalancer Names

		// LoadBalancerV2 Arns
		loadBalancersV2 := LoadBalancersV2{}
		if IsNukeable(loadBalancersV2.ResourceName(), nukeResourceTypes) {
			elbv2Arns, err := getAllElbv2Instances(sess, region, excludeAfter)
			if err != nil {
				if !IsErrIgnorable(loadBalancersV2.ResourceName(), ignoreErrResourceTypes) {
					return nil, errors.WithStackTrace(err)
				}
				logging.Logger.Warnf("Ignoring get resources error - %s - %s", loadBalancersV2.ResourceName(), err)
			} else {
				if len(elbv2Arns) > 0 {
					loadBalancersV2.Arns = awsgo.StringValueSlice(elbv2Arns)
					resourcesInRegion.Resources = append(resourcesInRegion.Resources, loadBalancersV2)
				}
			}
		}
		// End LoadBalancerV2 Arns

		// EC2 Instances
		ec2Instances := EC2Instances{}
		if IsNukeable(ec2Instances.ResourceName(), nukeResourceTypes) {
			instanceIds, err := getAllEc2Instances(sess, region, excludeAfter)
			if err != nil {
				if !IsErrIgnorable(ec2Instances.ResourceName(), ignoreErrResourceTypes) {
					return nil, errors.WithStackTrace(err)
				}
				logging.Logger.Warnf("Ignoring get resources error - %s - %s", ec2Instances.ResourceName(), err)
			} else {
				if len(instanceIds) > 0 {
					ec2Instances.InstanceIds = awsgo.StringValueSlice(instanceIds)
					resourcesInRegion.Resources = append(resourcesInRegion.Resources, ec2Instances)
				}
			}
		}
		// End EC2 Instances

		// EBS Volumes
		ebsVolumes := EBSVolumes{}
		if IsNukeable(ebsVolumes.ResourceName(), nukeResourceTypes) {
			volumeIds, err := getAllEbsVolumes(sess, region, excludeAfter)
			if err != nil {
				if !IsErrIgnorable(ebsVolumes.ResourceName(), ignoreErrResourceTypes) {
					return nil, errors.WithStackTrace(err)
				}
				logging.Logger.Warnf("Ignoring get resources error - %s - %s", ebsVolumes.ResourceName(), err)
			} else {
				if len(volumeIds) > 0 {
					ebsVolumes.VolumeIds = awsgo.StringValueSlice(volumeIds)
					resourcesInRegion.Resources = append(resourcesInRegion.Resources, ebsVolumes)
				}
			}
		}
		// End EBS Volumes

		// EIP Addresses
		eipAddresses := EIPAddresses{}
		if IsNukeable(eipAddresses.ResourceName(), nukeResourceTypes) {
			allocationIds, err := getAllEIPAddresses(sess, region, excludeAfter)
			if err != nil {
				if !IsErrIgnorable(eipAddresses.ResourceName(), ignoreErrResourceTypes) {
					return nil, errors.WithStackTrace(err)
				}
				logging.Logger.Warnf("Ignoring get resources error - %s - %s", eipAddresses.ResourceName(), err)
			} else {
				if len(allocationIds) > 0 {
					eipAddresses.AllocationIds = awsgo.StringValueSlice(allocationIds)
					resourcesInRegion.Resources = append(resourcesInRegion.Resources, eipAddresses)
				}
			}
		}
		// End EIP Addresses

		// AMIs
		amis := AMIs{}
		if IsNukeable(amis.ResourceName(), nukeResourceTypes) {
			imageIds, err := getAllAMIs(sess, region, excludeAfter)
			if err != nil {
				if !IsErrIgnorable(amis.ResourceName(), ignoreErrResourceTypes) {
					return nil, errors.WithStackTrace(err)
				}
				logging.Logger.Warnf("Ignoring get resources error - %s - %s", amis.ResourceName(), err)
			} else {
				if len(imageIds) > 0 {
					amis.ImageIds = awsgo.StringValueSlice(imageIds)
					resourcesInRegion.Resources = append(resourcesInRegion.Resources, amis)
				}
			}
		}
		// End AMIs

		// Snapshots
		snapshots := Snapshots{}
		if IsNukeable(snapshots.ResourceName(), nukeResourceTypes) {
			snapshotIds, err := getAllSnapshots(sess, region, excludeAfter)
			if err != nil {
				if !IsErrIgnorable(snapshots.ResourceName(), ignoreErrResourceTypes) {
					return nil, errors.WithStackTrace(err)
				}
				logging.Logger.Warnf("Ignoring get resources error - %s - %s", snapshots.ResourceName(), err)
			} else {
				if len(snapshotIds) > 0 {
					snapshots.SnapshotIds = awsgo.StringValueSlice(snapshotIds)
					resourcesInRegion.Resources = append(resourcesInRegion.Resources, snapshots)
				}
			}
		}
		// End Snapshots

		// ECS resources
		ecsServices := ECSServices{}
		if IsNukeable(ecsServices.ResourceName(), nukeResourceTypes) {
			clusterArns, err := getAllEcsClusters(sess)
			if err != nil {
				if !IsErrIgnorable(ecsServices.ResourceName(), ignoreErrResourceTypes) {
					return nil, errors.WithStackTrace(err)
				}
				logging.Logger.Warnf("Ignoring get resources error - %s - %s", ecsServices.ResourceName(), err)
			} else {
				if len(clusterArns) > 0 {
					serviceArns, serviceClusterMap, err := getAllEcsServices(sess, clusterArns, excludeAfter)
					if err != nil {
						return nil, errors.WithStackTrace(err)
					}
					ecsServices.Services = awsgo.StringValueSlice(serviceArns)
					ecsServices.ServiceClusterMap = serviceClusterMap
					resourcesInRegion.Resources = append(resourcesInRegion.Resources, ecsServices)
				}
			}
		}
		// End ECS resources

		// EKS resources
		eksClusters := EKSClusters{}
		if IsNukeable(eksClusters.ResourceName(), nukeResourceTypes) {
			if eksSupportedRegion(region) {
				eksClusterNames, err := getAllEksClusters(sess, excludeAfter)
				if err != nil {
					if !IsErrIgnorable(eksClusters.ResourceName(), ignoreErrResourceTypes) {
						return nil, errors.WithStackTrace(err)
					}
					logging.Logger.Warnf("Ignoring get resources error - %s - %s", eksClusters.ResourceName(), err)
				} else {
					if len(eksClusterNames) > 0 {
						eksClusters.Clusters = awsgo.StringValueSlice(eksClusterNames)
						resourcesInRegion.Resources = append(resourcesInRegion.Resources, eksClusters)
					}
				}
			}
		}
		// End EKS resources

		// RDS DB Instances
		dbInstances := DBInstances{}
		if IsNukeable(dbInstances.ResourceName(), nukeResourceTypes) {
			instanceNames, err := getAllRdsInstances(sess, excludeAfter)
			if err != nil {
				if !IsErrIgnorable(dbInstances.ResourceName(), ignoreErrResourceTypes) {
					return nil, errors.WithStackTrace(err)
				}
				logging.Logger.Warnf("Ignoring get resources error - %s - %s", dbInstances.ResourceName(), err)
			} else {
				if len(instanceNames) > 0 {
					dbInstances.InstanceNames = awsgo.StringValueSlice(instanceNames)
					resourcesInRegion.Resources = append(resourcesInRegion.Resources, dbInstances)
				}
			}
		}
		// End RDS DB Instances

		// RDS DB Clusters
		// These reference the Aurora Clusters, for the use it's the same resource (rds), but AWS
		// has different abstractions for each.
		dbClusters := DBClusters{}
		if IsNukeable(dbClusters.ResourceName(), nukeResourceTypes) {
			clustersNames, err := getAllRdsClusters(sess, excludeAfter)
			if err != nil {
				if !IsErrIgnorable(dbClusters.ResourceName(), ignoreErrResourceTypes) {
					return nil, errors.WithStackTrace(err)
				}
				logging.Logger.Warnf("Ignoring get resources error - %s - %s", dbClusters.ResourceName(), err)
			} else {
				if len(clustersNames) > 0 {
					dbClusters.InstanceNames = awsgo.StringValueSlice(clustersNames)
					resourcesInRegion.Resources = append(resourcesInRegion.Resources, dbClusters)
				}
			}
		}
		// End RDS DB Clusters

		// S3 Buckets
		s3Buckets := S3Buckets{}
		if IsNukeable(s3Buckets.ResourceName(), nukeResourceTypes) {
			var bucketNamesPerRegion map[string][]*string

			// AWS S3 buckets list operation lists all buckets irrespective of regions.
			// For each bucket we have to make a separate call to find the bucket region.
			// Hence for x buckets and a total of y target regions - we need to make:
			// (x + 1) * y calls i.e. 1 call to list all x buckets, x calls to find out
			// each bucket's region and repeat the process for each of the y regions.

			// getAllS3Buckets returns a map of regions to buckets and we call it only once -
			// thereby reducing total calls from (x + 1) * y to only (x + 1) for the first region -
			// followed by a cache lookup for rest of the regions.

			// Cache lookup to check if we already obtained bucket names per region
			bucketNamesPerRegion, ok := resourcesCache["S3"]

			if !ok {
				var err error
				bucketNamesPerRegion, err = getAllS3Buckets(sess, excludeAfter, targetRegions, "", s3Buckets.MaxConcurrentGetSize(), args.ConfigObj)
				if err != nil {
					if !IsErrIgnorable(s3Buckets.ResourceName(), ignoreErrResourceTypes) {
						return nil, errors.WithStackTrace(err)
					}
					logging.Logger.Warnf("Ignoring get resources error - %s - %s", s3Buckets.ResourceName(), err)
				} else {
					resourcesCache["S3"] = make(map[string][]*string)
					for bucketRegion, bucketName := range bucketNamesPerRegion {
						resourcesCache["S3"][bucketRegion] = bucketName
					}
				}
			}

			if len(bucketNamesPerRegion) > 0 {
				bucketNames := bucketNamesPerRegion[region]
				if len(bucketNamesPerRegion[region]) > 0 {
					s3Buckets.Names = aws.StringValueSlice(bucketNames)
					resourcesInRegion.Resources = append(resourcesInRegion.Resources, s3Buckets)
				}
			}
		}
		// End S3 Buckets

		if len(resourcesInRegion.Resources) > 0 {
			account.Resources[region] = resourcesInRegion
		}
		count++
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
		DBInstances{}.ResourceName(),
		S3Buckets{}.ResourceName(),
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

// IsErrIgnorable - Checks if we should ignore errors for resource type or not
func IsErrIgnorable(resourceType string, resourceTypes []string) bool {
	return collections.ListContainsElement(resourceTypes, "all") || collections.ListContainsElement(resourceTypes, resourceType)
}

// NukeAllResourcesArgs encapsulates arguments for NukeAllResources
type NukeAllResourcesArgs struct {
	Account                *AwsAccountResources        // resources to nuke
	Regions                []string                    // regions to nuke
	IgnoreErrResourceTypes []string                    // ignore errors encountered while nuking resources of this type
	RegionSessionMap       map[string]*session.Session // use pre-created per region sessions if passed in
}

// NukeAllResources - Nukes all aws resources
func NukeAllResources(args NukeAllResourcesArgs) error {
	account := args.Account
	regions := args.Regions
	ignoreErrResourceTypes := args.IgnoreErrResourceTypes
	regionSessionMap := args.RegionSessionMap

	for _, region := range regions {
		sess, ok := regionSessionMap[region]
		if !ok {
			var err error
			sess, err = session.NewSession(&awsgo.Config{
				Region: awsgo.String(region)},
			)
			if err != nil {
				return errors.WithStackTrace(err)
			}
		}

		resourcesInRegion := account.Resources[region]
		for _, resources := range resourcesInRegion.Resources {
			length := len(resources.ResourceIdentifiers())

			// Split api calls into batches
			logging.Logger.Infof("Terminating %d resources in batches", length)
			batches := split(resources.ResourceIdentifiers(), resources.MaxBatchSize())

			for i := 0; i < len(batches); i++ {
				batch := batches[i]
				if err := resources.Nuke(sess, batch); err != nil {
					// TODO: Figure out actual error type
					if strings.Contains(err.Error(), "RequestLimitExceeded") {
						logging.Logger.Info("Request limit reached. Waiting 1 minute before making new requests")
						time.Sleep(1 * time.Minute)
						continue
					}
					if IsErrIgnorable(resources.ResourceName(), ignoreErrResourceTypes) {
						logging.Logger.Warnf("Ignoring nuke resources error - %s - %s", resources.ResourceName(), err)
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
