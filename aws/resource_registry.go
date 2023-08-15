package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
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
		&IAMUsers{},
		&IAMGroups{},
		&IAMPolicies{},
		&IAMRoles{},
		&IAMServiceLinkedRoles{},
		&OIDCProviders{},
	}
}

func getRegisteredRegionalResources() []AwsResources {
	// Note: The order is important because it determines the order of nuking resources. Some resources need to
	// be deleted before others (Dependencies between resources exist). For example, we want to delete all EC2
	// instances before deleting the VPC.
	return []AwsResources{
		&AccessAnalyzer{},
		&ACM{},
		&ACMPCA{},
		&AMIs{},
		&ApiGateway{},
		&ApiGatewayV2{},
		&ASGroups{},
		&BackupVault{},
		&CloudtrailTrail{},
		&CloudWatchAlarms{},
		&CloudWatchDashboards{},
		&CloudWatchLogGroups{},
		&CodeDeployApplications{},
		&ConfigServiceRecorders{},
		&ConfigServiceRule{},
		&DynamoDB{},
		&EBSVolumes{},
		&EC2Instances{},
		&EC2DedicatedHosts{},
		&EC2KeyPairs{},
		&EC2VPCs{},
		&ECR{},
		&ECSClusters{},
		&ECSServices{},
		&ElasticFileSystem{},
		&EIPAddresses{},
		&EKSClusters{},
		&Elasticaches{},
		&ElasticacheParameterGroups{},
		&ElasticacheSubnetGroups{},
		&LoadBalancers{},
		&LoadBalancersV2{},
		&GuardDuty{},
		&KinesisStreams{},
		&KmsCustomerKeys{},
		&LambdaFunctions{},
		&LaunchConfigs{},
		&LaunchTemplates{},
		&MacieMember{},
		&NatGateways{},
		&OpenSearchDomains{},
		&DBInstances{},
		&DBSubnetGroups{},
		&DBClusters{},
		&RedshiftClusters{},
		&S3Buckets{},
		&SageMakerNotebookInstances{},
		&SecretsManagerSecrets{},
		&SecurityHub{},
		&Snapshots{},
		&SNSTopic{},
		&SqsQueue{},
		&TransitGatewaysVpcAttachment{},
		&TransitGatewaysRouteTables{},
		&TransitGateways{},
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
