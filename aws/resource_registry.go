package aws

import (
	"reflect"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gruntwork-io/cloud-nuke/aws/resources"
)

const Global = "global"

// GetAllRegisteredResources - returns a list of all registered resources without initialization.
// This is useful for listing all resources without initializing them.
func GetAllRegisteredResources() []*AwsResource {
	registeredResources := getRegisteredGlobalResources()
	registeredResources = append(registeredResources, getRegisteredRegionalResources()...)

	return toAwsResourcesPointer(registeredResources)
}

// GetAndInitRegisteredResources - returns a list of all registered resources with initialization.
func GetAndInitRegisteredResources(session aws.Config, region string) []*AwsResource {
	var registeredResources []AwsResource
	if region == Global {
		registeredResources = getRegisteredGlobalResources()
	} else {
		registeredResources = getRegisteredRegionalResources()
	}

	return initRegisteredResources(toAwsResourcesPointer(registeredResources), session, region)
}

// GetRegisteredGlobalResources - returns a list of registered global resources.
// Note: The order is important for IAM resources due to dependencies:
// 1. Users (removes itself from groups, detaches its policies)
// 2. Groups (removes remaining users, detaches its policies)
// 3. Roles (deletes instance profiles, detaches policies)
// 4. Service Linked Roles (special AWS-managed roles)
// 5. Instance Profiles (any remaining standalone profiles)
// 6. Policies (now fully detached from all entities)
func getRegisteredGlobalResources() []AwsResource {
	return []AwsResource{
		resources.NewCloudfrontDistributions(),
		resources.NewDBGlobalClusters(),
		// IAM deletion order: Users -> Groups -> Roles -> ServiceLinkedRoles -> InstanceProfiles -> Policies
		resources.NewIAMUsers(),
		resources.NewIAMGroups(),
		resources.NewIAMRoles(),
		resources.NewIAMServiceLinkedRoles(),
		resources.NewIAMInstanceProfiles(),
		resources.NewIAMPolicies(),
		resources.NewOIDCProviders(),
		resources.NewRoute53HostedZone(),
		resources.NewRoute53CidrCollections(),
		resources.NewRoute53TrafficPolicies(),
		resources.NewS3Buckets(),
	}
}

func getRegisteredRegionalResources() []AwsResource {
	// Note: The order is important because it determines the order of nuking resources. Some resources need to
	// be deleted before others (Dependencies between resources exist). For example, we want to delete all EC2
	// instances before deleting the VPC.
	return []AwsResource{
		resources.NewAccessAnalyzer(),
		resources.NewACM(),
		resources.NewACMPCA(),
		resources.NewAMIs(),
		resources.NewApiGateway(),
		resources.NewApiGatewayV2(),
		resources.NewASGroups(),
		resources.NewAppRunnerService(),
		resources.NewBackupVault(),
		resources.NewManagedPrometheus(),
		resources.NewGrafana(),
		resources.NewEventBridgeSchedule(),
		resources.NewEventBridgeScheduleGroup(),
		resources.NewEventBridgeArchive(),
		resources.NewEventBridgeRule(),
		resources.NewEventBridge(),
		resources.NewCloudtrailTrail(),
		resources.NewCloudFormationStacks(),
		resources.NewCloudWatchAlarms(),
		resources.NewCloudWatchDashboards(),
		resources.NewCloudWatchLogGroups(),
		resources.NewCloudMapServices(),
		resources.NewCloudMapNamespaces(),
		resources.NewCodeDeployApplications(),
		resources.NewConfigServiceRecorders(),
		resources.NewConfigServiceRules(),
		resources.NewDataSyncTask(),
		resources.NewDataSyncLocation(),
		resources.NewDynamoDB(),
		resources.NewEBSVolumes(),
		resources.NewEBApplications(),
		resources.NewEC2Instances(),
		resources.NewEC2DedicatedHosts(),
		resources.NewEC2KeyPairs(),
		resources.NewEC2PlacementGroups(),
		resources.NewTransitGateways(),
		resources.NewTransitGatewaysRouteTables(),
		// Note: nuking transitgateway vpc attachement before nuking the vpc since vpc could be associated with it.
		resources.NewTransitGatewayPeeringAttachment(),
		resources.NewTransitGatewaysVpcAttachment(),
		resources.NewEC2Endpoints(),
		resources.NewECR(),
		resources.NewECSClusters(),
		resources.NewECSServices(),
		resources.NewEgressOnlyInternetGateway(),
		resources.NewElasticFileSystem(),
		resources.NewEIPAddresses(),
		resources.NewEKSClusters(),
		resources.NewElasticCacheServerless(),
		resources.NewElasticaches(),
		resources.NewElasticacheParameterGroups(),
		resources.NewElasticacheSubnetGroups(),
		resources.NewLoadBalancers(),
		resources.NewLoadBalancersV2(),
		resources.NewGuardDuty(),
		resources.NewKinesisFirehose(),
		resources.NewKinesisStreams(),
		resources.NewKmsCustomerKeys(),
		resources.NewLambdaFunctions(),
		resources.NewLambdaLayers(),
		resources.NewLaunchConfigs(),
		resources.NewLaunchTemplates(),
		resources.NewMacieMember(),
		resources.NewMSKCluster(),
		resources.NewNatGateways(),
		resources.NewOpenSearchDomains(),
		resources.NewDBGlobalClusterMemberships(),
		resources.NewDBInstances(),
		resources.NewDBSubnetGroups(),
		resources.NewDBClusters(),
		resources.NewRdsProxy(),
		resources.NewRdsSnapshot(),
		resources.NewRdsParameterGroup(),
		resources.NewRedshiftClusters(),
		resources.NewRedshiftSnapshotCopyGrants(),
		resources.NewS3AccessPoints(),
		resources.NewS3ObjectLambdaAccessPoints(),
		resources.NewS3MultiRegionAccessPoints(),
		resources.NewSageMakerNotebookInstances(),
		resources.NewSageMakerStudio(),
		resources.NewSageMakerEndpoint(),
		resources.NewSageMakerEndpointConfig(),
		resources.NewSecretsManagerSecrets(),
		resources.NewSecurityHub(),
		resources.NewSesConfigurationSet(),
		resources.NewSesEmailTemplates(),
		resources.NewSesIdentities(),
		resources.NewSesReceiptRule(),
		resources.NewSesReceiptFilter(),
		resources.NewSnapshots(),
		resources.NewSNSTopic(),
		resources.NewSqsQueue(),
		resources.NewEC2IPAM(),
		resources.NewEC2IPAMScope(),
		resources.NewEC2IPAMResourceDiscovery(),
		resources.NewEC2IPAMPool(),
		resources.NewEC2IPAMByoasn(),
		resources.NewEC2IPAMCustomAllocation(),
		resources.NewNetworkFirewalls(),
		resources.NewNetworkFirewallPolicy(),
		resources.NewNetworkFirewallRuleGroup(),
		resources.NewNetworkFirewallTLSConfig(),
		resources.NewNetworkFirewallResourcePolicy(),
		resources.NewVPCLatticeServiceNetwork(),
		resources.NewVPCLatticeService(),
		resources.NewVPCLatticeTargetGroup(),
		// Note: VPC deletion order per AWS docs (https://docs.aws.amazon.com/vpc/latest/userguide/delete-vpc.html):
		// 1. Network Interfaces (ENIs must be deleted before security groups and subnets)
		resources.NewNetworkInterface(),
		// 2. Security Groups (must be deleted before VPC, after ENIs that reference them)
		resources.NewSecurityGroup(),
		// 3. Network ACLs (must be deleted before subnets)
		resources.NewNetworkACL(),
		// 4. Subnets (after ENIs, security groups, network ACLs)
		resources.NewEC2Subnet(),
		// 5. Internet Gateways (detach and delete before VPC)
		resources.NewInternetGateway(),
		// 6. VPCs (must be deleted last after all VPC-dependent resources)
		resources.NewEC2VPC(),
		// 7. DHCP Options (can be deleted after VPC since they're just disassociated)
		resources.NewEC2DhcpOptions(),
	}
}

func toAwsResourcesPointer(resources []AwsResource) []*AwsResource {
	var awsResourcePointers []*AwsResource
	for i := range resources {
		awsResourcePointers = append(awsResourcePointers, &resources[i])
	}

	return awsResourcePointers
}

func initRegisteredResources(resources []*AwsResource, session aws.Config, region string) []*AwsResource {
	for _, resource := range resources {
		(*resource).Init(session)

		// Note: only regional resources have the field `Region`, which is used for logging purposes only
		setRegionForRegionalResource(resource, region)
	}

	return resources
}

func setRegionForRegionalResource(regionResource *AwsResource, region string) {
	// Use reflection to set the Region field if the resource type has it
	resourceValue := reflect.ValueOf(*regionResource) // Dereference the pointer
	resourceValue = resourceValue.Elem()              // Get the underlying value
	regionField := resourceValue.FieldByName("Region")

	if regionField.IsValid() && regionField.CanSet() {
		// The field is valid and can be set
		regionField.SetString(region)
	}
}
