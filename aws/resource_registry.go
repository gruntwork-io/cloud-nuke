package aws

import (
	"reflect"

	"github.com/aws/aws-sdk-go/aws/session"
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
func GetAndInitRegisteredResources(session *session.Session, region string) []*AwsResource {
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
		&resources.IAMUsers{},
		&resources.IAMGroups{},
		&resources.IAMPolicies{},
		&resources.IAMRoles{},
		&resources.IAMServiceLinkedRoles{},
		&resources.OIDCProviders{},
	}
}

func getRegisteredRegionalResources() []AwsResource {
	// Note: The order is important because it determines the order of nuking resources. Some resources need to
	// be deleted before others (Dependencies between resources exist). For example, we want to delete all EC2
	// instances before deleting the VPC.
	return []AwsResource{
		&resources.AccessAnalyzer{},
		&resources.ACM{},
		&resources.ACMPCA{},
		&resources.AMIs{},
		&resources.ApiGateway{},
		&resources.ApiGatewayV2{},
		&resources.ASGroups{},
		&resources.BackupVault{},
		&resources.CloudtrailTrail{},
		&resources.CloudWatchAlarms{},
		&resources.CloudWatchDashboards{},
		&resources.CloudWatchLogGroups{},
		&resources.CodeDeployApplications{},
		&resources.ConfigServiceRecorders{},
		&resources.ConfigServiceRule{},
		&resources.DynamoDB{},
		&resources.EBSVolumes{},
		&resources.EBApplications{},
		&resources.EC2Instances{},
		&resources.EC2DedicatedHosts{},
		&resources.EC2KeyPairs{},
		&resources.TransitGateways{},
		&resources.TransitGatewaysRouteTables{},
		// Note: nuking transitgateway vpc attachement before nuking the vpc since vpc could be associated with it.
		&resources.TransitGatewaysVpcAttachment{},
		&resources.EC2Endpoints{},
		&resources.EC2VPCs{},
		// Note: nuking EC2 DHCP options after nuking EC2 VPC because DHCP options could be associated with VPCs.
		&resources.EC2DhcpOption{},
		&resources.ECR{},
		&resources.ECSClusters{},
		&resources.ECSServices{},
		&resources.EgressOnlyInternetGateway{},
		&resources.ElasticFileSystem{},
		&resources.EIPAddresses{},
		&resources.EKSClusters{},
		&resources.Elasticaches{},
		&resources.ElasticacheParameterGroups{},
		&resources.ElasticacheSubnetGroups{},
		&resources.LoadBalancers{},
		&resources.LoadBalancersV2{},
		&resources.GuardDuty{},
		&resources.KinesisStreams{},
		&resources.KmsCustomerKeys{},
		&resources.LambdaFunctions{},
		&resources.LambdaLayers{},
		&resources.LaunchConfigs{},
		&resources.LaunchTemplates{},
		&resources.MacieMember{},
		&resources.MSKCluster{},
		&resources.NatGateways{},
		&resources.OpenSearchDomains{},
		&resources.DBInstances{},
		&resources.DBSubnetGroups{},
		&resources.DBClusters{},
		&resources.RdsSnapshot{},
		&resources.RdsParameterGroup{},
		&resources.RedshiftClusters{},
		&resources.S3Buckets{},
		&resources.S3AccessPoint{},
		&resources.S3ObjectLambdaAccessPoint{},
		&resources.S3MultiRegionAccessPoint{},
		&resources.SageMakerNotebookInstances{},
		&resources.SecretsManagerSecrets{},
		&resources.SecurityHub{},
		&resources.SesConfigurationSet{},
		&resources.SesEmailTemplates{},
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
		&resources.Route53HostedZone{},
		&resources.Route53CidrCollection{},
		&resources.Route53TrafficPolicy{},
		&resources.InternetGateway{},
		&resources.NetworkInterface{},
		&resources.SecurityGroup{},
		&resources.NetworkACL{},
	}
}

func toAwsResourcesPointer(resources []AwsResource) []*AwsResource {
	var awsResourcePointers []*AwsResource
	for i := range resources {
		awsResourcePointers = append(awsResourcePointers, &resources[i])
	}

	return awsResourcePointers
}

func initRegisteredResources(resources []*AwsResource, session *session.Session, region string) []*AwsResource {
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
