package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/accessanalyzer"
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
)

// GetRegisteredRegionResource - returns a list of registered regional resources.
func GetRegisteredRegionResource(session *session.Session, region string) AwsRegionResource {
	// Note: The order is important because it determines the order of nuking resources. Some resources need to be deleted
	// before others (Dependencies between resources exist). For example, we want to delete all EC2
	// instances before deleting the VPC.
	return AwsRegionResource{
		Resources: []AwsResources{
			AccessAnalyzer{Client: accessanalyzer.New(session), Region: region},
			ACM{Client: acm.New(session), Region: region},
			ACMPCA{Client: acmpca.New(session), Region: region},
			AMIs{Client: ec2.New(session), Region: region},
			ApiGateway{Client: apigateway.New(session), Region: region},
			ApiGatewayV2{Client: apigatewayv2.New(session), Region: region},
			ASGroups{Client: autoscaling.New(session), Region: region},
			BackupVault{Client: backup.New(session), Region: region},
			CloudtrailTrail{Client: cloudtrail.New(session), Region: region},
			CloudWatchAlarms{Client: cloudwatch.New(session), Region: region},
			CloudWatchDashboards{Client: cloudwatch.New(session), Region: region},
			CloudWatchLogGroups{Client: cloudwatchlogs.New(session), Region: region},
			CodeDeployApplications{Client: codedeploy.New(session), Region: region},
			ConfigServiceRecorders{Client: configservice.New(session), Region: region},
			ConfigServiceRule{Client: configservice.New(session), Region: region},
			DynamoDB{Client: dynamodb.New(session), Region: region},
			EBSVolumes{Client: ec2.New(session), Region: region},
			EC2DedicatedHosts{Client: ec2.New(session), Region: region},
			EC2KeyPairs{Client: ec2.New(session), Region: region},
			EC2Instances{Client: ec2.New(session), Region: region},
			EC2VPCs{Client: ec2.New(session), Region: region},
			ECR{Client: ecr.New(session), Region: region},
			ECSClusters{Client: ecs.New(session), Region: region},
			ECSServices{Client: ecs.New(session), Region: region},
			ElasticFileSystem{Client: efs.New(session), Region: region},
			EIPAddresses{Client: ec2.New(session), Region: region},
			EKSClusters{Client: eks.New(session), Region: region},
			Elasticaches{Client: elasticache.New(session), Region: region},
			ElasticacheParameterGroups{Client: elasticache.New(session), Region: region},
			ElasticacheSubnetGroups{Client: elasticache.New(session), Region: region},
			LoadBalancers{Client: elb.New(session), Region: region},
			LoadBalancersV2{Client: elbv2.New(session), Region: region},
			GuardDuty{Client: guardduty.New(session), Region: region},
			KinesisStreams{Client: kinesis.New(session), Region: region},
			KmsCustomerKeys{Client: kms.New(session), Region: region},
			LambdaFunctions{Client: lambda.New(session), Region: region},
			LaunchConfigs{Client: autoscaling.New(session), Region: region},
			LaunchTemplates{Client: ec2.New(session), Region: region},
			MacieMember{Client: macie2.New(session), Region: region},
			NatGateways{Client: ec2.New(session), Region: region},
			OpenSearchDomains{Client: opensearchservice.New(session), Region: region},
			DBInstances{Client: rds.New(session), Region: region},
			DBClusters{Client: rds.New(session), Region: region},
			DBSubnetGroups{Client: rds.New(session), Region: region},
			RedshiftClusters{Client: redshift.New(session), Region: region},
			S3Buckets{Client: s3.New(session), Region: region},
			SageMakerNotebookInstances{Client: sagemaker.New(session), Region: region},
			SecretsManagerSecrets{Client: secretsmanager.New(session), Region: region},
			SecurityHub{Client: securityhub.New(session), Region: region},
			Snapshots{Client: ec2.New(session), Region: region},
			SNSTopic{Client: sns.New(session), Region: region},
			SqsQueue{Client: sqs.New(session), Region: region},
			TransitGateways{Client: ec2.New(session), Region: region},
		},
	}
}

// GetRegisteredGlobalResources - returns a list of registered global resources.
func GetRegisteredGlobalResources(session *session.Session) AwsRegionResource {
	resources := AwsRegionResource{
		Resources: []AwsResources{
			IAMUsers{Client: iam.New(session)},
			IAMGroups{Client: iam.New(session)},
			IAMPolicies{Client: iam.New(session)},
			IAMRoles{Client: iam.New(session)},
			IAMServiceLinkedRoles{Client: iam.New(session)},
			OIDCProviders{Client: iam.New(session)},
		},
	}

	return resources
}
