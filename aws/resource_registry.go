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
func getRegisteredGlobalResources() []AwsResource {
	return []AwsResource{
		&resources.DBGlobalClusters{},
		&resources.IAMUsers{},
		&resources.IAMGroups{},
		&resources.IAMPolicies{},
		&resources.IAMInstanceProfiles{},
		&resources.IAMRoles{},
		&resources.IAMServiceLinkedRoles{},
		&resources.OIDCProviders{},
		&resources.Route53HostedZone{},
		&resources.Route53CidrCollection{},
		&resources.Route53TrafficPolicy{},
	}
}

func getRegisteredRegionalResources() []AwsResource {
	// Note: The order is important because it determines the order of nuking resources. Some resources need to
	// be deleted before others (Dependencies between resources exist). For example, we want to delete all EC2
	// instances before deleting the VPC.
	return []AwsResource{
		resources.NewAccessAnalyzer(),
		&resources.ACM{},
		&resources.ACMPCA{},
		&resources.AMIs{},
		&resources.ApiGateway{},
		&resources.ApiGatewayV2{},
		&resources.ASGroups{},
		&resources.AppRunnerService{},
		&resources.BackupVault{},
		&resources.ManagedPrometheus{},
		&resources.Grafana{},
		resources.NewEventBridgeSchedule(),
		&resources.EventBridgeScheduleGroup{},
		&resources.EventBridgeArchive{},
		&resources.EventBridgeRule{},
		&resources.EventBridge{},
		&resources.CloudtrailTrail{},
		&resources.CloudFormationStacks{},
		&resources.CloudWatchAlarms{},
		resources.NewCloudWatchDashboards(),
		resources.NewCloudWatchLogGroups(),
		&resources.CloudMapServices{},
		&resources.CloudMapNamespaces{},
		&resources.CodeDeployApplications{},
		&resources.ConfigServiceRecorders{},
		&resources.ConfigServiceRule{},
		&resources.DataSyncTask{},
		&resources.DataSyncLocation{},
		&resources.DynamoDB{},
		&resources.EBSVolumes{},
		&resources.EBApplications{},
		&resources.EC2Instances{},
		&resources.EC2DedicatedHosts{},
		resources.NewEC2KeyPairs(),
		resources.NewEC2PlacementGroups(),
		&resources.TransitGateways{},
		&resources.TransitGatewaysRouteTables{},
		// Note: nuking transitgateway vpc attachement before nuking the vpc since vpc could be associated with it.
		&resources.TransitGatewayPeeringAttachment{},
		&resources.TransitGatewaysVpcAttachment{},
		&resources.EC2Endpoints{},
		&resources.ECR{},
		&resources.ECSClusters{},
		&resources.ECSServices{},
		&resources.EgressOnlyInternetGateway{},
		&resources.ElasticFileSystem{},
		&resources.EIPAddresses{},
		&resources.EKSClusters{},
		&resources.ElasticCacheServerless{},
		&resources.Elasticaches{},
		&resources.ElasticacheParameterGroups{},
		&resources.ElasticacheSubnetGroups{},
		&resources.LoadBalancers{},
		&resources.LoadBalancersV2{},
		&resources.GuardDuty{},
		resources.NewKinesisFirehose(),
		resources.NewKinesisStreams(),
		&resources.KmsCustomerKeys{},
		&resources.LambdaFunctions{},
		&resources.LambdaLayers{},
		resources.NewLaunchConfigs(),
		&resources.LaunchTemplates{},
		&resources.MacieMember{},
		&resources.MSKCluster{},
		&resources.NatGateways{},
		&resources.OpenSearchDomains{},
		&resources.DBGlobalClusterMemberships{},
		&resources.DBInstances{},
		&resources.DBSubnetGroups{},
		&resources.DBClusters{},
		&resources.RdsProxy{},
		&resources.RdsSnapshot{},
		&resources.RdsParameterGroup{},
		&resources.RedshiftClusters{},
		&resources.RedshiftSnapshotCopyGrants{},
		&resources.S3Buckets{},
		&resources.S3AccessPoint{},
		&resources.S3ObjectLambdaAccessPoint{},
		&resources.S3MultiRegionAccessPoint{},
		&resources.SageMakerNotebookInstances{},
		&resources.SageMakerStudio{},
		&resources.SageMakerEndpoint{},
		&resources.SecretsManagerSecrets{},
		&resources.SecurityHub{},
		&resources.SesConfigurationSet{},
		resources.NewSesEmailTemplates(),
		&resources.SesIdentities{},
		&resources.SesReceiptRule{},
		&resources.SesReceiptFilter{},
		&resources.Snapshots{},
		&resources.SNSTopic{},
		&resources.SqsQueue{},
		&resources.EC2IPAMs{},
		&resources.EC2IpamScopes{},
		&resources.EC2IPAMResourceDiscovery{},
		&resources.EC2IPAMPool{},
		&resources.EC2IPAMByoasn{},
		&resources.EC2IPAMCustomAllocation{},
		&resources.EC2Subnet{},
		&resources.InternetGateway{},
		&resources.NetworkInterface{},
		&resources.SecurityGroup{},
		&resources.NetworkACL{},
		&resources.NetworkFirewall{},
		&resources.NetworkFirewallPolicy{},
		&resources.NetworkFirewallRuleGroup{},
		&resources.NetworkFirewallTLSConfig{},
		&resources.NetworkFirewallResourcePolicy{},
		&resources.VPCLatticeServiceNetwork{},
		&resources.VPCLatticeService{},
		&resources.VPCLatticeTargetGroup{},
		// Note: VPCs must be deleted last after all resources that create network interfaces (EKS, ECS, etc.)
		&resources.EC2VPCs{},
		// Note: nuking EC2 DHCP options after nuking EC2 VPC because DHCP options could be associated with VPCs.
		&resources.EC2DhcpOption{},
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
