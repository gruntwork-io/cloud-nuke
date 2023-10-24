package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/cloud-nuke/aws/resources"
	"reflect"
)

const Global = "global"

// GetAllRegisteredResources - returns a list of all registered resources without initialization.
// This is useful for listing all resources without initializing them.
func GetAllRegisteredResources() []*AwsResources {
	resources := getRegisteredGlobalResources()
	resources = append(resources, getRegisteredRegionalResources()...)

	return toAwsResourcesPointer(resources)
}

// GetAndInitRegisteredResources - returns a list of all registered resources with initialization.
func GetAndInitRegisteredResources(session *session.Session, region string) []*AwsResources {
	var resources []AwsResources
	if region == Global {
		resources = getRegisteredGlobalResources()
	} else {
		resources = getRegisteredRegionalResources()
	}

	return initRegisteredResources(toAwsResourcesPointer(resources), session, region)
}

// GetRegisteredGlobalResources - returns a list of registered global resources.
func getRegisteredGlobalResources() []AwsResources {
	return []AwsResources{
		&resources.IAMUsers{},
		&resources.IAMGroups{},
		&resources.IAMPolicies{},
		&resources.IAMRoles{},
		&resources.IAMServiceLinkedRoles{},
		&resources.OIDCProviders{},
	}
}

func getRegisteredRegionalResources() []AwsResources {
	// Note: The order is important because it determines the order of nuking resources. Some resources need to
	// be deleted before others (Dependencies between resources exist). For example, we want to delete all EC2
	// instances before deleting the VPC.
	return []AwsResources{
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
		&resources.EC2Instances{},
		&resources.EC2DedicatedHosts{},
		&resources.EC2KeyPairs{},
		&resources.EC2VPCs{},
		// Note: nuking EC2 DHCP options after nuking EC2 VPC because DHCP options could be associated with VPCs.
		&resources.EC2DhcpOption{},
		&resources.ECR{},
		&resources.ECSClusters{},
		&resources.ECSServices{},
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
		&resources.LaunchConfigs{},
		&resources.LaunchTemplates{},
		&resources.MacieMember{},
		&resources.NatGateways{},
		&resources.OpenSearchDomains{},
		&resources.DBInstances{},
		&resources.DBSubnetGroups{},
		&resources.DBClusters{},
		&resources.RdsSnapshot{},
		&resources.RedshiftClusters{},
		&resources.S3Buckets{},
		&resources.SageMakerNotebookInstances{},
		&resources.SecretsManagerSecrets{},
		&resources.SecurityHub{},
		&resources.Snapshots{},
		&resources.SNSTopic{},
		&resources.SqsQueue{},
		&resources.TransitGatewaysVpcAttachment{},
		&resources.TransitGatewaysRouteTables{},
		&resources.TransitGateways{},
	}
}

func toAwsResourcesPointer(resources []AwsResources) []*AwsResources {
	var awsResourcePointers []*AwsResources
	for i := range resources {
		awsResourcePointers = append(awsResourcePointers, &resources[i])
	}

	return awsResourcePointers
}

func initRegisteredResources(resources []*AwsResources, session *session.Session, region string) []*AwsResources {
	for _, resource := range resources {
		(*resource).Init(session)

		// Note: only regional resources have the field `Region`, which is used for logging purposes only
		setRegionForRegionalResource(resource, region)
	}

	return resources
}

func setRegionForRegionalResource(regionResource *AwsResources, region string) {
	// Use reflection to set the Region field if the resource type has it
	resourceValue := reflect.ValueOf(*regionResource) // Dereference the pointer
	resourceValue = resourceValue.Elem()              // Get the underlying value
	regionField := resourceValue.FieldByName("Region")

	if regionField.IsValid() && regionField.CanSet() {
		// The field is valid and can be set
		regionField.SetString(region)
	}
}
