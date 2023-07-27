package aws

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/acmpca"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/apigatewayv2"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/backup"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/codedeploy"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/guardduty"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/macie2"
	"github.com/aws/aws-sdk-go/service/opensearchservice"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sagemaker"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/externalcreds"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/progressbar"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/collections"
	"github.com/gruntwork-io/go-commons/errors"
)

// OptInNotRequiredRegions contains all regions that are enabled by default on new AWS accounts
// Beginning in Spring 2019, AWS requires new regions to be explicitly enabled
// See https://aws.amazon.com/blogs/security/setting-permissions-to-enable-accounts-for-upcoming-aws-regions/
var OptInNotRequiredRegions = []string{
	"eu-north-1",
	"ap-south-1",
	"eu-west-3",
	"eu-west-2",
	"eu-west-1",
	"ap-northeast-3",
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

// GovCloudRegions contains all of the U.S. GovCloud regions. In accounts with GovCloud enabled, these are the
// only available regions.
var GovCloudRegions = []string{
	"us-gov-east-1",
	"us-gov-west-1",
}

const (
	GlobalRegion string = "global"
)

func newSession(region string) *session.Session {
	return externalcreds.Get(region)
}

// Try a describe regions command with the most likely enabled regions
func retryDescribeRegions() (*ec2.DescribeRegionsOutput, error) {
	regionsToTry := append(OptInNotRequiredRegions, GovCloudRegions...)
	for _, region := range regionsToTry {
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
	return getRandomRegionWithExclusions([]string{})
}

// getRandomRegionWithExclusions - return random from enabled regions, excluding regions from the argument
func getRandomRegionWithExclusions(regionsToExclude []string) (string, error) {
	allRegions, err := GetEnabledRegions()
	if err != nil {
		return "", errors.WithStackTrace(err)
	}
	rand.Seed(time.Now().UnixNano())

	// exclude from "allRegions"
	exclusions := make(map[string]string)
	for _, region := range regionsToExclude {
		exclusions[region] = region
	}
	// filter regions
	var updatedRegions []string
	for _, region := range allRegions {
		_, excluded := exclusions[region]
		if !excluded {
			updatedRegions = append(updatedRegions, region)
		}
	}
	randIndex := rand.Intn(len(updatedRegions))
	logging.Logger.Debugf("Random region chosen: %s", updatedRegions[randIndex])
	return updatedRegions[randIndex], nil
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
		chunks = append(chunks, identifiers[:])
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

// GetAllResources - Lists all aws resources
func GetAllResources(targetRegions []string, excludeAfter time.Time, resourceTypes []string, configObj config.Config, allowDeleteUnaliasedKeys bool) (*AwsAccountResources, error) {
	account := AwsAccountResources{
		Resources: make(map[string]AwsRegionResource),
	}

	count := 1
	totalRegions := len(targetRegions)
	resourcesCache := map[string]map[string][]*string{}

	defaultRegion := targetRegions[0]
	for _, region := range targetRegions {
		// The "global" region case is handled outside this loop
		if region == GlobalRegion {
			continue
		}

		logging.Logger.Debugf("Checking region [%d/%d]: %s", count, totalRegions, region)

		cloudNukeSession := newSession(region)
		stsService := sts.New(cloudNukeSession)
		resp, err := stsService.GetCallerIdentity(&sts.GetCallerIdentityInput{})
		if err == nil {
			telemetry.SetAccountId(*resp.Account)
		}

		resourcesInRegion := AwsRegionResource{}

		// The order in which resources are nuked is important
		// because of dependencies between resources

		// ACMPCA arns
		acmpca := ACMPCA{
			Client: acmpca.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(acmpca.ResourceName(), resourceTypes) {
			start := time.Now()
			arns, err := acmpca.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve ACMPCAs",
					ResourceType: acmpca.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing ACMPA arns",
			}, map[string]interface{}{
				"recordCount": len(arns),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(arns) > 0 {
				acmpca.ARNs = awsgo.StringValueSlice(arns)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, acmpca)
			}
		}
		// End ACMPCA arns

		// ASG Names
		asGroups := ASGroups{
			Client: autoscaling.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(asGroups.ResourceName(), resourceTypes) {
			start := time.Now()
			groupNames, err := asGroups.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve Auto-Scaling Groups",
					ResourceType: asGroups.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing ASGs",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(groupNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(groupNames) > 0 {
				asGroups.GroupNames = awsgo.StringValueSlice(groupNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, asGroups)
			}
		}
		// End ASG Names

		// Launch Configuration Names
		configs := LaunchConfigs{}
		if IsNukeable(configs.ResourceName(), resourceTypes) {
			start := time.Now()
			configNames, err := getAllLaunchConfigurations(cloudNukeSession, region, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve Launch configurations",
					ResourceType: configs.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Launch Configurations",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(configNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(configNames) > 0 {
				configs.LaunchConfigurationNames = awsgo.StringValueSlice(configNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, configs)
			}
		}
		// End Launch Configuration Names

		// Launch Template Names
		templates := LaunchTemplates{
			Client: ec2.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(templates.ResourceName(), resourceTypes) {
			start := time.Now()
			templateNames, err := getAllLaunchTemplates(cloudNukeSession, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve Launch templates",
					ResourceType: templates.ResourceName(),
				}
				report.RecordError(ge)
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Launch Templates",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(templateNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(templateNames) > 0 {
				templates.LaunchTemplateNames = awsgo.StringValueSlice(templateNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, templates)
			}
		}
		// End Launch Template Names

		// LoadBalancers Names
		loadBalancers := LoadBalancers{
			Client: elb.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(loadBalancers.ResourceName(), resourceTypes) {
			start := time.Now()
			elbNames, err := getAllElbInstances(cloudNukeSession, region, excludeAfter)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve load balancers",
					ResourceType: loadBalancers.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing ELBs",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(elbNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(elbNames) > 0 {
				loadBalancers.Names = awsgo.StringValueSlice(elbNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, loadBalancers)
			}
		}
		// End LoadBalancer Names

		// LoadBalancerV2 Arns
		loadBalancersV2 := LoadBalancersV2{
			Client: elbv2.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(loadBalancersV2.ResourceName(), resourceTypes) {
			start := time.Now()
			elbv2Arns, err := getAllElbv2Instances(cloudNukeSession, region, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve load balancers v2",
					ResourceType: loadBalancersV2.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing ELBV2s",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(elbv2Arns),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(elbv2Arns) > 0 {
				loadBalancersV2.Arns = awsgo.StringValueSlice(elbv2Arns)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, loadBalancersV2)
			}
		}
		// End LoadBalancerV2 Arns

		// SQS Queues
		sqsQueue := SqsQueue{
			Client: sqs.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(sqsQueue.ResourceName(), resourceTypes) {
			start := time.Now()
			queueUrls, err := sqsQueue.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve SQS queues",
					ResourceType: sqsQueue.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing SQS Queues",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(queueUrls),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(queueUrls) > 0 {
				sqsQueue.QueueUrls = awsgo.StringValueSlice(queueUrls)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, sqsQueue)
			}
		}
		// End SQS Queue

		// TransitGatewayVpcAttachment
		transitGatewayVpcAttachments := TransitGatewaysVpcAttachment{
			Client: ec2.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(transitGatewayVpcAttachments.ResourceName(), resourceTypes) {
			start := time.Now()
			transitGatewayVpcAttachmentIds, err := transitGatewayVpcAttachments.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Could not retrieve Transit Gateway attachments",
					ResourceType: transitGatewayVpcAttachments.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Transit Gateway VPC Attachments",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(transitGatewayVpcAttachmentIds),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(transitGatewayVpcAttachmentIds) > 0 {
				transitGatewayVpcAttachments.Ids = awsgo.StringValueSlice(transitGatewayVpcAttachmentIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, transitGatewayVpcAttachments)
			}
		}
		// End TransitGatewayVpcAttachment

		// TransitGatewayRouteTable
		transitGatewayRouteTables := TransitGatewaysRouteTables{
			Client: ec2.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(transitGatewayRouteTables.ResourceName(), resourceTypes) {
			start := time.Now()
			transitGatewayRouteTableIds, err := transitGatewayRouteTables.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve Transit Gateway route tables",
					ResourceType: transitGatewayRouteTables.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Transit Gateway Route Tables",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(transitGatewayRouteTableIds),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(transitGatewayRouteTableIds) > 0 {
				transitGatewayRouteTables.Ids = awsgo.StringValueSlice(transitGatewayRouteTableIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, transitGatewayRouteTables)
			}
		}
		// End TransitGatewayRouteTable

		// TransitGateway
		transitGateways := TransitGateways{
			Client: ec2.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(transitGateways.ResourceName(), resourceTypes) {
			start := time.Now()
			transitGatewayIds, err := transitGateways.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve Transit Gateways",
					ResourceType: transitGateways.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Transit Gateway Instances",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(transitGatewayIds),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(transitGatewayIds) > 0 {
				transitGateways.Ids = awsgo.StringValueSlice(transitGatewayIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, transitGateways)
			}
		}
		// End TransitGateway

		// NATGateway
		natGateways := NatGateways{
			Client: ec2.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(natGateways.ResourceName(), resourceTypes) {
			start := time.Now()
			ngwIDs, err := getAllNatGateways(cloudNukeSession, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve NAT Gateways",
					ResourceType: natGateways.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing NAT Gateways",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(ngwIDs),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(ngwIDs) > 0 {
				natGateways.NatGatewayIDs = awsgo.StringValueSlice(ngwIDs)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, natGateways)
			}
		}
		// End NATGateway

		// OpenSearch Domains
		domains := OpenSearchDomains{
			Client: opensearchservice.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(domains.ResourceName(), resourceTypes) {
			start := time.Now()
			domainNames, err := domains.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve OpenSearch Domains",
					ResourceType: domains.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Opensearch Domains",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(domainNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(domainNames) > 0 {
				domains.DomainNames = awsgo.StringValueSlice(domainNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, domains)
			}
		}
		// End OpenSearchDomains

		// EC2 Instances
		ec2Instances := EC2Instances{
			Client: ec2.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(ec2Instances.ResourceName(), resourceTypes) {
			start := time.Now()
			instanceIds, err := getAllEc2Instances(cloudNukeSession, region, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve EC2 instances",
					ResourceType: ec2Instances.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing EC2 Instances",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(instanceIds),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(instanceIds) > 0 {
				ec2Instances.InstanceIds = awsgo.StringValueSlice(instanceIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, ec2Instances)
			}
		}
		// End EC2 Instances

		// EC2 Dedicated Hosts
		ec2DedicatedHosts := EC2DedicatedHosts{
			Client: ec2.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(ec2DedicatedHosts.ResourceName(), resourceTypes) {
			start := time.Now()
			hostIds, err := getAllEc2DedicatedHosts(cloudNukeSession, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve EC2 dedicated hosts",
					ResourceType: ec2DedicatedHosts.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing EC2 Dedicated Hosts",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(hostIds),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(hostIds) > 0 {
				ec2DedicatedHosts.HostIds = awsgo.StringValueSlice(hostIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, ec2DedicatedHosts)
			}
		}

		// End EC2 Dedicated Hosts

		// EBS Volumes
		ebsVolumes := EBSVolumes{
			Client: ec2.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(ebsVolumes.ResourceName(), resourceTypes) {
			start := time.Now()
			volumeIds, err := ebsVolumes.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve EBS volumes",
					ResourceType: ebsVolumes.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing EBS Volumes",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(volumeIds),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(volumeIds) > 0 {
				ebsVolumes.VolumeIds = awsgo.StringValueSlice(volumeIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, ebsVolumes)
			}
		}
		// End EBS Volumes

		// EIP Addresses
		eipAddresses := EIPAddresses{
			Client: ec2.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(eipAddresses.ResourceName(), resourceTypes) {
			start := time.Now()
			allocationIds, err := getAllEIPAddresses(cloudNukeSession, region, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve EIP addresses",
					ResourceType: eipAddresses.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing EIPs",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(allocationIds),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(allocationIds) > 0 {
				eipAddresses.AllocationIds = awsgo.StringValueSlice(allocationIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, eipAddresses)
			}
		}
		// End EIP Addresses

		// AMIs
		amis := AMIs{
			Client: ec2.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(amis.ResourceName(), resourceTypes) {
			start := time.Now()
			imageIds, err := amis.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve AMIs",
					ResourceType: amis.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing AMIs",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(imageIds),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(imageIds) > 0 {
				amis.ImageIds = awsgo.StringValueSlice(imageIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, amis)
			}
		}
		// End AMIs

		// Snapshots
		snapshots := Snapshots{
			Client: ec2.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(snapshots.ResourceName(), resourceTypes) {
			start := time.Now()
			snapshotIds, err := snapshots.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve Snapshots",
					ResourceType: snapshots.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Snapshots",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(snapshotIds),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(snapshotIds) > 0 {
				snapshots.SnapshotIds = awsgo.StringValueSlice(snapshotIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, snapshots)
			}
		}
		// End Snapshots

		// ECS resources
		ecsServices := ECSServices{
			Client: ecs.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(ecsServices.ResourceName(), resourceTypes) {
			start := time.Now()
			clusterArns, err := getAllEcsClusters(cloudNukeSession)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve ECS clusters",
					ResourceType: ecsServices.ResourceName(),
				}
				report.RecordError(ge)
			}
			if len(clusterArns) > 0 {
				serviceArns, serviceClusterMap, err := getAllEcsServices(cloudNukeSession, clusterArns, excludeAfter, configObj)
				if err != nil {
					return nil, errors.WithStackTrace(err)
				}
				ecsServices.Services = awsgo.StringValueSlice(serviceArns)
				ecsServices.ServiceClusterMap = serviceClusterMap
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, ecsServices)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing ECS Services",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(clusterArns),
				"actionTime":  time.Since(start).Seconds(),
			})
		}

		ecsClusters := ECSClusters{
			Client: ecs.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(ecsClusters.ResourceName(), resourceTypes) {
			start := time.Now()
			ecsClusterArns, err := getAllEcsClustersOlderThan(cloudNukeSession, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve ECS clusters",
					ResourceType: ecsClusters.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing ECS Clusters",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(ecsClusterArns),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(ecsClusterArns) > 0 {
				ecsClusters.ClusterArns = awsgo.StringValueSlice(ecsClusterArns)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, ecsClusters)
			}
		}
		// End ECS resources

		// EKS resources
		eksClusters := EKSClusters{
			Client: eks.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(eksClusters.ResourceName(), resourceTypes) {
			start := time.Now()
			eksClusterNames, err := eksClusters.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve EKS clusters",
					ResourceType: eksClusters.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing EKS Clusters",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(eksClusterNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(eksClusterNames) > 0 {
				eksClusters.Clusters = awsgo.StringValueSlice(eksClusterNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, eksClusters)
			}
		}
		// End EKS resources

		// RDS DB Instances
		dbInstances := DBInstances{
			Client: rds.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(dbInstances.ResourceName(), resourceTypes) {
			start := time.Now()
			instanceNames, err := dbInstances.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve DB instances",
					ResourceType: dbInstances.ResourceName(),
				}
				report.RecordError(ge)
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing RDS Instances",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(instanceNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(instanceNames) > 0 {
				dbInstances.InstanceNames = awsgo.StringValueSlice(instanceNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, dbInstances)
			}
		}
		// End RDS DB Instances

		// RDS DB Subnet Groups
		dbSubnetGroups := DBSubnetGroups{
			Client: rds.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(dbSubnetGroups.ResourceName(), resourceTypes) {
			start := time.Now()

			// Note: the `DescribeDBSubnetGroups` API response does not contain any information
			// about when the subnet group was created, so we cannot apply the `excludeAfter` filter
			subnetGroups, err := getAllRdsDbSubnetGroups(cloudNukeSession, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve DB subnet groups",
					ResourceType: dbSubnetGroups.ResourceName(),
				}
				report.RecordError(ge)
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing RDS Subnet Groups",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(subnetGroups),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(subnetGroups) > 0 {
				dbSubnetGroups.InstanceNames = awsgo.StringValueSlice(subnetGroups)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, dbSubnetGroups)
			}
		}
		// End RDS DB Subnet Groups

		// RDS DB Clusters
		// These reference the Aurora Clusters, for the use it's the same resource (rds), but AWS
		// has different abstractions for each.
		dbClusters := DBClusters{
			Client: rds.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(dbClusters.ResourceName(), resourceTypes) {
			start := time.Now()
			clustersNames, err := dbClusters.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve DB clusters",
					ResourceType: dbClusters.ResourceName(),
				}
				report.RecordError(ge)
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing RDS Clusters",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(clustersNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(clustersNames) > 0 {
				dbClusters.InstanceNames = awsgo.StringValueSlice(clustersNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, dbClusters)
			}
		}
		// End RDS DB Clusters

		// Backup Vaults
		backupVault := BackupVault{
			Client: backup.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(backupVault.ResourceName(), resourceTypes) {
			start := time.Now()
			backupVaultNames, err := backupVault.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve backup vaults",
					ResourceType: backupVault.ResourceName(),
				}
				report.RecordError(ge)
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing backup vaults",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(backupVaultNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(backupVaultNames) > 0 {
				backupVault.Names = awsgo.StringValueSlice(backupVaultNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, backupVault)
			}
		}
		// End backup vaults

		// Lambda Functions
		lambdaFunctions := LambdaFunctions{
			Client: lambda.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(lambdaFunctions.ResourceName(), resourceTypes) {
			start := time.Now()
			lambdaFunctionNames, err := getAllLambdaFunctions(cloudNukeSession, excludeAfter, configObj, lambdaFunctions.MaxBatchSize())
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve Lambda functions",
					ResourceType: lambdaFunctions.ResourceName(),
				}
				report.RecordError(ge)
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Lambda Functions",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(lambdaFunctionNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(lambdaFunctionNames) > 0 {
				lambdaFunctions.LambdaFunctionNames = awsgo.StringValueSlice(lambdaFunctionNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, lambdaFunctions)
			}
		}
		// End Lambda Functions

		// Secrets Manager Secrets
		secretsManagerSecrets := SecretsManagerSecrets{
			Client: secretsmanager.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(secretsManagerSecrets.ResourceName(), resourceTypes) {
			start := time.Now()
			secrets, err := getAllSecretsManagerSecrets(cloudNukeSession, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve Secrets managers entries",
					ResourceType: secretsManagerSecrets.ResourceName(),
				}
				report.RecordError(ge)
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Secrets Manager Secrets",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(secrets),
				"actionTime":  time.Since(start).Seconds(),
			})

			if len(secrets) > 0 {
				secretsManagerSecrets.SecretIDs = awsgo.StringValueSlice(secrets)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, secretsManagerSecrets)
			}
		}
		// End Secrets Manager Secrets

		// AccessAnalyzer
		accessAnalyzer := AccessAnalyzer{}
		if IsNukeable(accessAnalyzer.ResourceName(), resourceTypes) {
			start := time.Now()
			analyzerNames, err := getAllAccessAnalyzers(cloudNukeSession, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve Access analyzers",
					ResourceType: accessAnalyzer.ResourceName(),
				}
				report.RecordError(ge)
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Access Analyzers",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(analyzerNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(analyzerNames) > 0 {
				accessAnalyzer.AnalyzerNames = awsgo.StringValueSlice(analyzerNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, accessAnalyzer)
			}
		}
		// End AccessAnalyzer

		// CloudWatchDashboard
		cloudwatchDashboards := CloudWatchDashboards{
			Client: cloudwatch.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(cloudwatchDashboards.ResourceName(), resourceTypes) {
			start := time.Now()
			cwdbNames, err := cloudwatchDashboards.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve CloudWatch dashboards",
					ResourceType: cloudwatchDashboards.ResourceName(),
				}
				report.RecordError(ge)
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Cloudwatch Dashboards",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(cwdbNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(cwdbNames) > 0 {
				cloudwatchDashboards.DashboardNames = awsgo.StringValueSlice(cwdbNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, cloudwatchDashboards)
			}
		}
		// End CloudWatchDashboard

		// CloudWatchLogGroup
		cloudwatchLogGroups := CloudWatchLogGroups{
			Client: cloudwatchlogs.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(cloudwatchLogGroups.ResourceName(), resourceTypes) {
			start := time.Now()
			lgNames, err := cloudwatchLogGroups.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve CloudWatch log groups",
					ResourceType: cloudwatchLogGroups.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Cloudwatch Log Groups",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(lgNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(lgNames) > 0 {
				cloudwatchLogGroups.Names = awsgo.StringValueSlice(lgNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, cloudwatchLogGroups)
			}
		}
		// End CloudWatchLogGroup

		// S3 Buckets
		s3Buckets := S3Buckets{
			Client: s3.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(s3Buckets.ResourceName(), resourceTypes) {
			start := time.Now()
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
				bucketNamesPerRegion, err = getAllS3Buckets(
					cloudNukeSession,
					excludeAfter,
					targetRegions,
					"",
					s3Buckets.MaxConcurrentGetSize(),
					configObj,
				)
				if err != nil {
					ge := report.GeneralError{
						Error:        err,
						Description:  "Unable to retrieve S3 buckets",
						ResourceType: s3Buckets.ResourceName(),
					}
					report.RecordError(ge)
				}

				resourcesCache["S3"] = make(map[string][]*string)

				for bucketRegion := range bucketNamesPerRegion {
					resourcesCache["S3"][bucketRegion] = bucketNamesPerRegion[bucketRegion]
				}
			}

			bucketNames, ok := resourcesCache["S3"][region]

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing S3 Buckets",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(bucketNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if ok && len(bucketNames) > 0 {
				s3Buckets.Names = aws.StringValueSlice(bucketNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, s3Buckets)
			}
		}
		// End S3 Buckets

		DynamoDB := DynamoDB{
			Client: dynamodb.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(DynamoDB.ResourceName(), resourceTypes) {
			start := time.Now()
			tablenames, err := DynamoDB.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve Dynamo DB tables",
					ResourceType: DynamoDB.ResourceName(),
				}
				report.RecordError(ge)
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing DynamoDB Tables",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(tablenames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(tablenames) > 0 {
				DynamoDB.DynamoTableNames = awsgo.StringValueSlice(tablenames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, DynamoDB)
			}
		}
		// End Dynamo DB tables

		// EC2 VPCS
		ec2Vpcs := EC2VPCs{
			Client: ec2.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(ec2Vpcs.ResourceName(), resourceTypes) {
			start := time.Now()
			vpcids, vpcs, err := getAllVpcs(cloudNukeSession, region, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve VPCs",
					ResourceType: ec2Vpcs.ResourceName(),
				}
				report.RecordError(ge)
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing EC2 VPCs",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(vpcids),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(vpcids) > 0 {
				ec2Vpcs.VPCIds = awsgo.StringValueSlice(vpcids)
				ec2Vpcs.VPCs = vpcs
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, ec2Vpcs)
			}
		}
		// End EC2 VPCS

		// Start EC2 KeyPairs
		KeyPairs := EC2KeyPairs{}
		if IsNukeable(KeyPairs.ResourceName(), resourceTypes) {
			start := time.Now()
			keyPairIds, err := getAllEc2KeyPairs(cloudNukeSession, excludeAfter, configObj)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing EC2 Keypairs",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(keyPairIds),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(keyPairIds) > 0 {
				KeyPairs.KeyPairIds = awsgo.StringValueSlice(keyPairIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, KeyPairs)
			}
		}
		// End EC2 KeyPairs

		// Elasticaches
		elasticaches := Elasticaches{
			Client: elasticache.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(elasticaches.ResourceName(), resourceTypes) {
			start := time.Now()
			clusterIds, err := getAllElasticacheClusters(cloudNukeSession, region, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve Elasticaches",
					ResourceType: elasticaches.ResourceName(),
				}
				report.RecordError(ge)
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Elasticache Clusters",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(clusterIds),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(clusterIds) > 0 {
				elasticaches.ClusterIds = awsgo.StringValueSlice(clusterIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, elasticaches)
			}
		}
		// End Elasticaches

		// Elasticache Parameter Groups
		elasticacheParameterGroups := ElasticacheParameterGroups{
			Client: elasticache.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(elasticacheParameterGroups.ResourceName(), resourceTypes) {
			start := time.Now()
			groupNames, err := getAllElasticacheParameterGroups(cloudNukeSession, region, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve Elasticache Parameter Groups",
					ResourceType: elasticacheParameterGroups.ResourceName(),
				}
				report.RecordError(ge)
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Elasticache Parameter Groups",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(groupNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(groupNames) > 0 {
				elasticacheParameterGroups.GroupNames = awsgo.StringValueSlice(groupNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, elasticacheParameterGroups)
			}
		}
		// End Elasticache Parameter Groups

		// Elasticache Subnet Groups
		elasticacheSubnetGroups := ElasticacheSubnetGroups{
			Client: elasticache.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(elasticacheSubnetGroups.ResourceName(), resourceTypes) {
			start := time.Now()
			groupNames, err := getAllElasticacheSubnetGroups(cloudNukeSession, region, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve Elasticache Subnet Groups",
					ResourceType: elasticacheSubnetGroups.ResourceName(),
				}
				report.RecordError(ge)
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Elasticache Subnet Groups",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(groupNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(groupNames) > 0 {
				elasticacheSubnetGroups.GroupNames = awsgo.StringValueSlice(groupNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, elasticacheSubnetGroups)
			}
		}
		// End Elasticache Subnet Groups

		// KMS Customer managed keys
		customerKeys := KmsCustomerKeys{
			Client: kms.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(customerKeys.ResourceName(), resourceTypes) {
			start := time.Now()
			keys, aliases, err := getAllKmsUserKeys(cloudNukeSession, customerKeys.MaxBatchSize(), excludeAfter, configObj, allowDeleteUnaliasedKeys)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve KMS customer keys",
					ResourceType: customerKeys.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing KMS Keys",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(keys),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(keys) > 0 {
				customerKeys.KeyAliases = aliases
				customerKeys.KeyIds = awsgo.StringValueSlice(keys)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, customerKeys)
			}

		}
		// End KMS Customer managed keys

		// GuardDuty detectors
		guardDutyDetectors := GuardDuty{
			Client: guardduty.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(guardDutyDetectors.ResourceName(), resourceTypes) {
			start := time.Now()
			detectors, err := getAllGuardDutyDetectors(cloudNukeSession, excludeAfter, configObj, guardDutyDetectors.MaxBatchSize())
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve GuardDuty detectors",
					ResourceType: guardDutyDetectors.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Guard Duty Detectors",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(detectors),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(detectors) > 0 {
				guardDutyDetectors.detectorIds = detectors
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, guardDutyDetectors)
			}
		}
		// End GuardDuty detectors

		// Macie member accounts
		macieAccounts := MacieMember{
			Client: macie2.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(macieAccounts.ResourceName(), resourceTypes) {
			start := time.Now()
			// Unfortunately, the Macie API doesn't provide the metadata information we'd need to implement the excludeAfter or configObj patterns
			accountIds, err := getMacie(cloudNukeSession, excludeAfter)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve Macie member accounts",
					ResourceType: macieAccounts.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing MACIE Member Accounts",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(accountIds),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(accountIds) > 0 {
				macieAccounts.AccountIds = accountIds
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, macieAccounts)
			}

		}
		// End Macie member accounts

		// Start SageMaker Notebook Instances
		notebookInstances := SageMakerNotebookInstances{
			Client: sagemaker.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(notebookInstances.ResourceName(), resourceTypes) {
			start := time.Now()
			instances, err := notebookInstances.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve sagemaker notebook instances",
					ResourceType: notebookInstances.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Sagemaker Notebook Instances",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(instances),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(instances) > 0 {
				notebookInstances.InstanceNames = awsgo.StringValueSlice(instances)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, notebookInstances)
			}
		}
		// End SageMaker Notebook Instances

		// Kinesis Streams
		kinesisStreams := KinesisStreams{
			Client: kinesis.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(kinesisStreams.ResourceName(), resourceTypes) {
			start := time.Now()
			streams, err := getAllKinesisStreams(cloudNukeSession, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve kinesis streams",
					ResourceType: kinesisStreams.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Kinesis Streams",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(streams),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(streams) > 0 {
				kinesisStreams.Names = awsgo.StringValueSlice(streams)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, kinesisStreams)
			}
		}
		// End Kinesis Streams

		// Redshift Clusters
		redshiftClusters := RedshiftClusters{
			Client: redshift.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(redshiftClusters.ResourceName(), resourceTypes) {
			start := time.Now()
			clusters, err := redshiftClusters.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve redshift clusters",
					ResourceType: redshiftClusters.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Redshift Clusters",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(clusters),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(clusters) > 0 {
				redshiftClusters.ClusterIdentifiers = awsgo.StringValueSlice(clusters)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, redshiftClusters)
			}
		}
		// End Redshift Clusters

		// API Gateways (v1)
		apiGateways := ApiGateway{
			Client: apigateway.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(apiGateways.ResourceName(), resourceTypes) {
			start := time.Now()
			gatewayIds, err := apiGateways.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve API gateways v1",
					ResourceType: apiGateways.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing API Gateways",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(gatewayIds),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(gatewayIds) > 0 {
				apiGateways.Ids = awsgo.StringValueSlice(gatewayIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, apiGateways)
			}
		}
		// End API Gateways (v1)

		// API Gateways (v2)
		apiGatewaysV2 := ApiGatewayV2{
			Client: apigatewayv2.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(apiGatewaysV2.ResourceName(), resourceTypes) {
			start := time.Now()
			gatewayV2Ids, err := apiGatewaysV2.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve API gateways v2",
					ResourceType: apiGatewaysV2.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing API Gateway V2s",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(gatewayV2Ids),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(gatewayV2Ids) > 0 {
				apiGatewaysV2.Ids = awsgo.StringValueSlice(gatewayV2Ids)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, apiGatewaysV2)
			}
		}
		// End API Gateways (v2)

		// Elastic FileSystems (efs)
		elasticFileSystems := ElasticFileSystem{
			Client: efs.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(elasticFileSystems.ResourceName(), resourceTypes) {
			start := time.Now()
			elasticFileSystemsIds, err := getAllElasticFileSystems(cloudNukeSession, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve Elastic FileSystems",
					ResourceType: elasticFileSystems.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Elastic File Systems",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(elasticFileSystemsIds),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(elasticFileSystemsIds) > 0 {
				elasticFileSystems.Ids = awsgo.StringValueSlice(elasticFileSystemsIds)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, elasticFileSystems)
			}
		}
		// End Elastic FileSystems (efs)

		// SNS Topics
		snsTopics := SNSTopic{
			Client: sns.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(snsTopics.ResourceName(), resourceTypes) {
			start := time.Now()
			snsTopicArns, err := snsTopics.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve SNS topics",
					ResourceType: snsTopics.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing SNS Topics",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(snsTopicArns),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(snsTopicArns) > 0 {
				snsTopics.Arns = awsgo.StringValueSlice(snsTopicArns)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, snsTopics)
			}
		}
		// End SNS Topics

		// Cloudtrail Trails
		cloudtrailTrails := CloudtrailTrail{
			Client: cloudtrail.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(cloudtrailTrails.ResourceName(), resourceTypes) {
			start := time.Now()
			cloudtrailArns, err := cloudtrailTrails.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve Cloudtrail trails",
					ResourceType: cloudtrailTrails.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing CloudTrails",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(cloudtrailArns),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(cloudtrailArns) > 0 {
				cloudtrailTrails.Arns = awsgo.StringValueSlice(cloudtrailArns)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, cloudtrailTrails)
			}
		}
		// End Cloudtrail Trails

		// ECR Repositories
		ecrRepositories := ECR{
			Client: ecr.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(ecrRepositories.ResourceName(), resourceTypes) {
			start := time.Now()
			ecrRepositoryArns, err := getAllECRRepositories(cloudNukeSession, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve ECR repositories",
					ResourceType: ecrRepositories.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing ECR Repos",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(ecrRepositoryArns),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(ecrRepositoryArns) > 0 {
				ecrRepositories.RepositoryNames = ecrRepositoryArns
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, ecrRepositories)
			}
		}
		// End ECR Repositories

		// Config Service Rules
		configServiceRules := ConfigServiceRule{
			Client: configservice.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(configServiceRules.ResourceName(), resourceTypes) {
			start := time.Now()
			configServiceRuleNames, err := configServiceRules.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve Config service rules",
					ResourceType: configServiceRules.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Config Service rules",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(configServiceRuleNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(configServiceRuleNames) > 0 {
				configServiceRules.RuleNames = awsgo.StringValueSlice(configServiceRuleNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, configServiceRules)
			}
		}
		// End Config service rules

		// Config Service recorders
		configServiceRecorders := ConfigServiceRecorders{
			Client: configservice.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(configServiceRecorders.ResourceName(), resourceTypes) {
			start := time.Now()
			configServiceRecorderNames, err := configServiceRecorders.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve Config service recorders",
					ResourceType: configServiceRecorders.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Config Service rules",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(configServiceRecorderNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(configServiceRecorderNames) > 0 {
				configServiceRecorders.RecorderNames = awsgo.StringValueSlice(configServiceRecorderNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, configServiceRecorders)
			}
		}
		// End Config service recorders

		// CloudWatchAlarm
		cloudwatchAlarms := CloudWatchAlarms{
			Client: cloudwatch.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(cloudwatchAlarms.ResourceName(), resourceTypes) {
			start := time.Now()
			cwalNames, err := cloudwatchAlarms.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve CloudWatch alarms",
					ResourceType: cloudwatchAlarms.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Cloudwatch Alarms",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(cwalNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(cwalNames) > 0 {
				cloudwatchAlarms.AlarmNames = awsgo.StringValueSlice(cwalNames)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, cloudwatchAlarms)
			}
		}
		// End CloudWatchAlarm

		// Security Hub
		securityHub := SecurityHub{
			Client: securityhub.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(securityHub.ResourceName(), resourceTypes) {
			start := time.Now()
			hubArns, err := securityHub.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve security hub status",
					ResourceType: securityHub.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing Security Hub",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(hubArns),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(hubArns) > 0 {
				securityHub.HubArns = awsgo.StringValueSlice(hubArns)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, securityHub)
			}
		}
		// End Security Hub

		// CodeDeploy Applications
		codeDeployApplications := CodeDeployApplications{
			Client: codedeploy.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(codeDeployApplications.ResourceName(), resourceTypes) {
			start := time.Now()
			applications, err := codeDeployApplications.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve CodeDeploy applications",
					ResourceType: codeDeployApplications.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing CodeDeploy Applications",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(applications),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(applications) > 0 {
				codeDeployApplications.AppNames = awsgo.StringValueSlice(applications)
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, codeDeployApplications)
			}
		}
		// End CodeDeploy Applications

		// ACM
		acm := ACM{
			Client: acm.New(cloudNukeSession),
			Region: region,
		}
		if IsNukeable(acm.ResourceName(), resourceTypes) {
			start := time.Now()
			acmArns, err := acm.getAll(configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve ACM status",
					ResourceType: acm.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing ACM Certificates",
			}, map[string]interface{}{
				"region":      region,
				"recordCount": len(acmArns),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(acmArns) > 0 {
				acm.ARNs = acmArns
				resourcesInRegion.Resources = append(resourcesInRegion.Resources, acm)
			}
		}
		// End ACM

		if len(resourcesInRegion.Resources) > 0 {
			account.Resources[region] = resourcesInRegion
		}
		count++

	}

	// Global Resources - These resources are global and do not belong to a specific region
	// Only process them if the global region was not explicitly excluded
	if collections.ListContainsElement(targetRegions, GlobalRegion) {
		logging.Logger.Debugf("Checking region [%d/%d]: %s", count, totalRegions, GlobalRegion)

		// As there is no actual region named global we have to pick a valid one just to create the session
		session := newSession(defaultRegion)
		globalResources := AwsRegionResource{}

		// IAM Users
		iamUsers := IAMUsers{
			Client: iam.New(session),
		}
		if IsNukeable(iamUsers.ResourceName(), resourceTypes) {
			start := time.Now()
			userNames, err := getAllIamUsers(session, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve IAM users",
					ResourceType: iamUsers.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing IAM Users",
			}, map[string]interface{}{
				"region":      "global",
				"recordCount": len(userNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(userNames) > 0 {
				iamUsers.UserNames = awsgo.StringValueSlice(userNames)
				globalResources.Resources = append(globalResources.Resources, iamUsers)
			}
		}
		// End IAM Users

		// IAM Groups
		iamGroups := IAMGroups{
			Client: iam.New(session),
		}
		if IsNukeable(iamGroups.ResourceName(), resourceTypes) {
			start := time.Now()
			groupNames, err := getAllIamGroups(session, excludeAfter, configObj)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing IAM Groups",
			}, map[string]interface{}{
				"region":      "global",
				"recordCount": len(groupNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(groupNames) > 0 {
				iamGroups.GroupNames = awsgo.StringValueSlice(groupNames)
				globalResources.Resources = append(globalResources.Resources, iamGroups)
			}
		}
		// END IAM Groups

		// IAM Policies
		iamPolicies := IAMPolicies{
			Client: iam.New(session),
		}
		if IsNukeable(iamPolicies.ResourceName(), resourceTypes) {
			start := time.Now()
			policyArns, err := getAllLocalIamPolicies(session, excludeAfter, configObj)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing IAM Policies",
			}, map[string]interface{}{
				"region":      "global",
				"recordCount": len(policyArns),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(policyArns) > 0 {
				iamPolicies.PolicyArns = awsgo.StringValueSlice(policyArns)
				globalResources.Resources = append(globalResources.Resources, iamPolicies)
			}
		}
		// End IAM Policies

		// IAM OpenID Connect Providers
		oidcProviders := OIDCProviders{
			Client: iam.New(session),
		}
		if IsNukeable(oidcProviders.ResourceName(), resourceTypes) {
			start := time.Now()
			providerARNs, err := getAllOIDCProviders(session, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve OIDC providers",
					ResourceType: oidcProviders.ResourceName(),
				}
				report.RecordError(ge)
			}

			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing OIDC Providers",
			}, map[string]interface{}{
				"region":      "global",
				"recordCount": len(providerARNs),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(providerARNs) > 0 {
				oidcProviders.ProviderARNs = awsgo.StringValueSlice(providerARNs)
				globalResources.Resources = append(globalResources.Resources, oidcProviders)
			}
		}
		// End IAM OpenIDConnectProviders

		// IAM Roles
		iamRoles := IAMRoles{
			Client: iam.New(session),
		}
		if IsNukeable(iamRoles.ResourceName(), resourceTypes) {
			start := time.Now()
			roleNames, err := getAllIamRoles(session, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve IAM roles",
					ResourceType: iamRoles.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing IAM Roles",
			}, map[string]interface{}{
				"region":      "global",
				"recordCount": len(roleNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(roleNames) > 0 {
				iamRoles.RoleNames = awsgo.StringValueSlice(roleNames)
				globalResources.Resources = append(globalResources.Resources, iamRoles)
			}
		}
		// End IAM Roles

		// IAM Service Linked Roles
		iamServiceLinkedRoles := IAMServiceLinkedRoles{
			Client: iam.New(session),
		}
		if IsNukeable(iamServiceLinkedRoles.ResourceName(), resourceTypes) {
			start := time.Now()
			roleNames, err := getAllIamServiceLinkedRoles(session, excludeAfter, configObj)
			if err != nil {
				ge := report.GeneralError{
					Error:        err,
					Description:  "Unable to retrieve IAM roles",
					ResourceType: iamServiceLinkedRoles.ResourceName(),
				}
				report.RecordError(ge)
			}
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Done Listing IAM Service Linked Roles",
			}, map[string]interface{}{
				"region":      "global",
				"recordCount": len(roleNames),
				"actionTime":  time.Since(start).Seconds(),
			})
			if len(roleNames) > 0 {
				iamServiceLinkedRoles.RoleNames = awsgo.StringValueSlice(roleNames)
				globalResources.Resources = append(globalResources.Resources, iamServiceLinkedRoles)
			}
		}
		// End IAM Service Linked Roles

		if len(globalResources.Resources) > 0 {
			account.Resources[GlobalRegion] = globalResources
		}
	}

	return &account, nil
}

// ListResourceTypes - Returns list of resources which can be passed to --resource-type
func ListResourceTypes() []string {
	resourceTypes := []string{
		ACMPCA{}.ResourceName(),
		ASGroups{}.ResourceName(),
		LaunchConfigs{}.ResourceName(),
		LoadBalancers{}.ResourceName(),
		LoadBalancersV2{}.ResourceName(),
		SqsQueue{}.ResourceName(),
		TransitGatewaysVpcAttachment{}.ResourceName(),
		TransitGatewaysRouteTables{}.ResourceName(),
		TransitGateways{}.ResourceName(),
		EC2Instances{}.ResourceName(),
		EC2DedicatedHosts{}.ResourceName(),
		EBSVolumes{}.ResourceName(),
		EIPAddresses{}.ResourceName(),
		AMIs{}.ResourceName(),
		Snapshots{}.ResourceName(),
		ECSClusters{}.ResourceName(),
		ECSServices{}.ResourceName(),
		EKSClusters{}.ResourceName(),
		DBInstances{}.ResourceName(),
		LambdaFunctions{}.ResourceName(),
		S3Buckets{}.ResourceName(),
		IAMUsers{}.ResourceName(),
		IAMRoles{}.ResourceName(),
		IAMGroups{}.ResourceName(),
		IAMPolicies{}.ResourceName(),
		IAMServiceLinkedRoles{}.ResourceName(),
		SecretsManagerSecrets{}.ResourceName(),
		NatGateways{}.ResourceName(),
		OpenSearchDomains{}.ResourceName(),
		CloudWatchDashboards{}.ResourceName(),
		AccessAnalyzer{}.ResourceName(),
		DynamoDB{}.ResourceName(),
		EC2VPCs{}.ResourceName(),
		Elasticaches{}.ResourceName(),
		ElasticacheParameterGroups{}.ResourceName(),
		ElasticacheSubnetGroups{}.ResourceName(),
		OIDCProviders{}.ResourceName(),
		KmsCustomerKeys{}.ResourceName(),
		CloudWatchLogGroups{}.ResourceName(),
		GuardDuty{}.ResourceName(),
		MacieMember{}.ResourceName(),
		SageMakerNotebookInstances{}.ResourceName(),
		KinesisStreams{}.ResourceName(),
		RedshiftClusters{}.ResourceName(),
		ApiGateway{}.ResourceName(),
		ApiGatewayV2{}.ResourceName(),
		ElasticFileSystem{}.ResourceName(),
		SNSTopic{}.ResourceName(),
		CloudtrailTrail{}.ResourceName(),
		EC2KeyPairs{}.ResourceName(),
		ECR{}.ResourceName(),
		LaunchTemplates{}.ResourceName(),
		ConfigServiceRule{}.ResourceName(),
		ConfigServiceRecorders{}.ResourceName(),
		SecurityHub{}.ResourceName(),
		CloudWatchAlarms{}.ResourceName(),
		ACM{}.ResourceName(),
		CodeDeployApplications{}.ResourceName(),
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

func nukeAllResourcesInRegion(account *AwsAccountResources, region string, session *session.Session) {
	resourcesInRegion := account.Resources[region]

	for _, resources := range resourcesInRegion.Resources {
		length := len(resources.ResourceIdentifiers())

		// Split api calls into batches
		logging.Logger.Debugf("Terminating %d resources in batches", length)
		batches := split(resources.ResourceIdentifiers(), resources.MaxBatchSize())

		for i := 0; i < len(batches); i++ {
			batch := batches[i]
			if err := resources.Nuke(session, batch); err != nil {
				// TODO: Figure out actual error type
				if strings.Contains(err.Error(), "RequestLimitExceeded") {
					logging.Logger.Debug(
						"Request limit reached. Waiting 1 minute before making new requests",
					)
					time.Sleep(1 * time.Minute)
					continue
				}

				// We're only interested in acting on Rate limit errors - no other error should prevent further processing
				// of the current job.Since we handle each individual resource deletion error within its own resource-specific code,
				// we can safely discard this error
				_ = err
			}

			if i != len(batches)-1 {
				logging.Logger.Debug("Sleeping for 10 seconds before processing next batch...")
				time.Sleep(10 * time.Second)
			}
		}
	}
}

// StartProgressBarWithLength - Starts the progress bar with the correct number of items
func StartProgressBarWithLength(length int) {
	// Update the progress bar to have the correct width based on the total number of unique resource targteds
	progressbar.WithTotal(length)
	p := progressbar.GetProgressbar()
	p.Start()
}

// NukeAllResources - Nukes all aws resources
func NukeAllResources(account *AwsAccountResources, regions []string) error {
	// Set the progressbar width to the total number of nukeable resources found
	// across all regions
	StartProgressBarWithLength(account.TotalResourceCount())

	telemetry.TrackEvent(commonTelemetry.EventContext{
		EventName: "Begin nuking resources",
	}, map[string]interface{}{})

	defaultRegion := regions[0]
	for _, region := range regions {
		// region that will be used to create a session
		sessionRegion := region

		// As there is no actual region named global we have to pick a valid one just to create the session
		if region == GlobalRegion {
			sessionRegion = defaultRegion
		}

		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Creating session for region",
		}, map[string]interface{}{
			"region": region,
		})
		session := newSession(sessionRegion)
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Nuking Region",
		}, map[string]interface{}{
			"region":        region,
			"resourceCount": len(account.Resources[region].Resources),
		})

		// We intentionally do not handle an error returned from this method, because we collect individual errors
		// on per-resource basis via the report package's Record method. In the run report displayed at the end of
		// a cloud-nuke run, we show exactly which resources deleted cleanly and which encountered errors
		nukeAllResourcesInRegion(account, region, session)
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Done Nuking Region",
		}, map[string]interface{}{
			"region":        region,
			"resourceCount": len(account.Resources[region].Resources),
		})
	}

	return nil
}
