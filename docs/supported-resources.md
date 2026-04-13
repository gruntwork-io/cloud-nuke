# Supported Resources

cloud-nuke supports inspecting and deleting the following AWS resources. The **CLI ID** column is the value you pass to `--resource-type`.

| CLI ID | Resource |
|---|---|
| `access-analyzer` | IAM Access Analyzer |
| `acm` | ACM Certificate |
| `acmpca` | ACM Private CA |
| `ami` | EC2 AMI |
| `api-gateway` | API Gateway (v1) |
| `api-gateway-v2` | API Gateway (v2) |
| `app-runner-service` | App Runner Service |
| `asg` | Auto Scaling Group |
| `backup-vault` | Backup Vault |
| `cloudformation-stack` | CloudFormation Stack |
| `cloudfront-distribution` | CloudFront Distribution |
| `cloudmap-namespace` | Cloud Map Namespace |
| `cloudmap-service` | Cloud Map Service |
| `cloudtrail` | CloudTrail Trail |
| `cloudwatch-alarm` | CloudWatch Alarm |
| `cloudwatch-dashboard` | CloudWatch Dashboard |
| `cloudwatch-loggroup` | CloudWatch Log Group |
| `codedeploy-application` | CodeDeploy Application |
| `config-recorders` | Config Service Recorder |
| `config-rules` | Config Service Rule |
| `data-pipeline` | Data Pipeline |
| `data-sync-location` | DataSync Location |
| `data-sync-task` | DataSync Task |
| `dynamodb` | DynamoDB Table |
| `ebs` | EBS Volume |
| `ebs-snapshot` | EBS Snapshot |
| `ec2` | EC2 Instance |
| `ec2-dedicated-hosts` | EC2 Dedicated Host |
| `ec2-dhcp-option` | EC2 DHCP Option Set |
| `ec2-endpoint` | EC2 VPC Endpoint |
| `ec2-keypairs` | EC2 Key Pair |
| `ec2-placement-groups` | EC2 Placement Group |
| `ec2-subnet` | EC2 Subnet |
| `ecr` | ECR Repository |
| `ecs-cluster` | ECS Cluster |
| `ecs-service` | ECS Service |
| `efs` | EFS File System |
| `egress-only-internet-gateway` | Egress Only Internet Gateway |
| `eip` | Elastic IP |
| `eks-cluster` | EKS Cluster |
| `elastic-beanstalk` | Elastic Beanstalk Application |
| `elasticache` | ElastiCache Cluster |
| `elasticache-parameter-group` | ElastiCache Parameter Group |
| `elasticache-serverless` | ElastiCache Serverless Cluster |
| `elasticache-subnet-group` | ElastiCache Subnet Group |
| `elb` | Classic Load Balancer |
| `elbv2` | Application/Network Load Balancer |
| `event-bridge` | EventBridge Bus |
| `event-bridge-archive` | EventBridge Archive |
| `event-bridge-rule` | EventBridge Rule |
| `event-bridge-schedule` | EventBridge Schedule |
| `event-bridge-schedule-group` | EventBridge Schedule Group |
| `grafana` | Grafana Workspace |
| `guard-duty` | GuardDuty Detector |
| `iam-group` | IAM Group |
| `iam-instance-profile` | IAM Instance Profile |
| `iam-policy` | IAM Policy |
| `iam-role` | IAM Role |
| `iam-service-linked-role` | IAM Service-linked Role |
| `iam-user` | IAM User |
| `internet-gateway` | Internet Gateway |
| `ipam` | EC2 IPAM |
| `ipam-byoasn` | EC2 IPAM BYOASN |
| `ipam-custom-allocation` | EC2 IPAM Custom Allocation |
| `ipam-pool` | EC2 IPAM Pool |
| `ipam-resource-discovery` | EC2 IPAM Resource Discovery |
| `ipam-scope` | EC2 IPAM Scope |
| `kinesis-firehose` | Kinesis Firehose |
| `kinesis-stream` | Kinesis Stream |
| `kms-customer-key` | KMS Customer Managed Key |
| `lambda` | Lambda Function |
| `lambda-layer` | Lambda Layer |
| `launch-configuration` | Launch Configuration |
| `launch-template` | Launch Template |
| `macie-member` | Macie Member Account |
| `managed-prometheus` | Managed Prometheus Workspace |
| `mq-broker` | Amazon MQ Broker |
| `msk-cluster` | MSK Cluster |
| `nat-gateway` | NAT Gateway |
| `network-acl` | Network ACL |
| `network-firewall` | Network Firewall |
| `network-firewall-policy` | Network Firewall Policy |
| `network-firewall-resource-policy` | Network Firewall Resource Policy |
| `network-firewall-rule-group` | Network Firewall Rule Group |
| `network-firewall-tls-config` | Network Firewall TLS Config |
| `network-interface` | Network Interface |
| `oidc-provider` | OIDC Provider |
| `opensearch-domain` | OpenSearch Domain |
| `rds-cluster` | RDS DB Cluster |
| `rds-global-cluster` | RDS Global Cluster |
| `rds-global-cluster-membership` | RDS Global Cluster Membership |
| `rds-instance` | RDS DB Instance (incl. Neptune, DocumentDB) |
| `rds-parameter-group` | RDS Parameter Group |
| `rds-proxy` | RDS Proxy |
| `rds-cluster-snapshot` | RDS Cluster Snapshot |
| `rds-snapshot` | RDS Snapshot |
| `rds-subnet-group` | RDS Subnet Group |
| `redshift` | Redshift Cluster |
| `redshift-snapshot-copy-grant` | Redshift Snapshot Copy Grant |
| `resource-share` | RAM Resource Share |
| `route-table` | Route Table |
| `route53-cidr-collection` | Route53 CIDR Collection |
| `route53-hosted-zone` | Route53 Hosted Zone |
| `route53-traffic-policy` | Route53 Traffic Policy |
| `s3` | S3 Bucket |
| `s3-access-point` | S3 Access Point |
| `s3-multi-region-access-point` | S3 Multi Region Access Point |
| `s3-object-lambda-access-point` | S3 Object Lambda Access Point |
| `sagemaker-endpoint` | SageMaker Endpoint |
| `sagemaker-endpoint-config` | SageMaker Endpoint Configuration |
| `sagemaker-notebook-instance` | SageMaker Notebook Instance |
| `sagemaker-studio` | SageMaker Studio Domain |
| `secrets-manager` | Secrets Manager Secret |
| `security-group` | Security Group |
| `security-hub` | Security Hub |
| `ses-configuration-set` | SES Configuration Set |
| `ses-email-template` | SES Email Template |
| `ses-identity` | SES Identity |
| `ses-receipt-filter` | SES Receipt Filter |
| `ses-receipt-rule-set` | SES Receipt Rule Set |
| `sns-topic` | SNS Topic |
| `sqs` | SQS Queue |
| `ssm-parameter` | SSM Parameter Store Parameter |
| `transit-gateway` | Transit Gateway |
| `transit-gateway-attachment` | Transit Gateway VPC Attachment |
| `transit-gateway-peering-attachment` | Transit Gateway Peering Attachment |
| `transit-gateway-route-table` | Transit Gateway Route Table |
| `vpc` | VPC |
| `vpc-lattice-service` | VPC Lattice Service |
| `vpc-lattice-service-network` | VPC Lattice Service Network |
| `vpc-lattice-target-group` | VPC Lattice Target Group |
| `vpc-peering-connection` | VPC Peering Connection |

> **WARNING:** The RDS APIs also interact with Neptune and DocumentDB resources. Running `cloud-nuke aws --resource-type rds-instance` without a config file will remove any Neptune and DocumentDB resources in the account.

> **NOTE:** Resources created by AWS Backup are managed by AWS Backup and cannot be deleted through standard API calls. These resources are tagged by AWS Backup and are automatically filtered out by cloud-nuke.

## Config Support Matrix

This table shows which filtering features are supported for each resource type in the [config file](configuration.md).

| Resource Type | Config Key | names_regex | time | tags | timeout |
|---|---|---|---|---|---|
| access-analyzer | AccessAnalyzer | ✓ | ✓ | ✓ | ✓ |
| acm | ACM | ✓ | ✓ | ✓ | ✓ |
| acmpca | ACMPCA | | ✓ | ✓ | ✓ |
| ami | AMI | ✓ | ✓ | ✓ | ✓ |
| api-gateway | APIGateway | ✓ | ✓ | ✓ | ✓ |
| api-gateway-v2 | APIGatewayV2 | ✓ | ✓ | ✓ | ✓ |
| app-runner-service | AppRunnerService | ✓ | ✓ | ✓ | ✓ |
| asg | AutoScalingGroup | ✓ | ✓ | ✓ | ✓ |
| backup-vault | BackupVault | ✓ | ✓ | ✓ | ✓ |
| cloudformation-stack | CloudFormationStack | ✓ | ✓ | ✓ | ✓ |
| cloudfront-distribution | CloudFrontDistribution | ✓ | | ✓ | ✓ |
| cloudmap-namespace | CloudMapNamespace | ✓ | ✓ | ✓ | ✓ |
| cloudmap-service | CloudMapService | ✓ | ✓ | ✓ | ✓ |
| cloudtrail | CloudTrailTrail | ✓ | | ✓ | ✓ |
| cloudwatch-alarm | CloudWatchAlarm | ✓ | ✓ | ✓ | ✓ |
| cloudwatch-dashboard | CloudWatchDashboard | ✓ | ✓ | | ✓ |
| cloudwatch-loggroup | CloudWatchLogGroup | ✓ | ✓ | ✓ | ✓ |
| codedeploy-application | CodeDeployApplications | ✓ | ✓ | ✓ | ✓ |
| config-recorders | ConfigServiceRecorder | ✓ | | | ✓ |
| config-rules | ConfigServiceRule | ✓ | | ✓ | ✓ |
| data-pipeline | DataPipeline | ✓ | ✓ | ✓ | ✓ |
| data-sync-location | DataSyncLocation | ✓ | | ✓ | ✓ |
| data-sync-task | DataSyncTask | ✓ | | ✓ | ✓ |
| dynamodb | DynamoDB | ✓ | ✓ | ✓ | ✓ |
| ebs | EBSVolume | ✓ | ✓ | ✓ | ✓ |
| ebs-snapshot | Snapshots | | ✓ | ✓ | ✓ |
| ec2 | EC2 | ✓ | ✓ | ✓ | ✓ |
| ec2-dedicated-hosts | EC2DedicatedHosts | ✓ | ✓ | ✓ | ✓ |
| ec2-dhcp-option | EC2DHCPOption | | | ✓ | ✓ |
| ec2-endpoint | EC2Endpoint | ✓ | ✓ | ✓ | ✓ |
| ec2-keypairs | EC2KeyPairs | ✓ | ✓ | ✓ | ✓ |
| ec2-placement-groups | EC2PlacementGroups | ✓ | ✓ | ✓ | ✓ |
| ec2-subnet | EC2Subnet | ✓ | ✓ | ✓ | |
| ecr | ECRRepository | ✓ | ✓ | ✓ | ✓ |
| ecs-cluster | ECSCluster | ✓ | ✓ | ✓ | ✓ |
| ecs-service | ECSService | ✓ | ✓ | ✓ | ✓ |
| efs | ElasticFileSystem | ✓ | ✓ | ✓ | ✓ |
| egress-only-internet-gateway | EgressOnlyInternetGateway | ✓ | | ✓ | ✓ |
| eip | ElasticIP | ✓ | ✓ | ✓ | ✓ |
| eks-cluster | EKSCluster | ✓ | ✓ | ✓ | ✓ |
| elastic-beanstalk | ElasticBeanstalk | ✓ | ✓ | ✓ | ✓ |
| elasticache | ElastiCache | ✓ | ✓ | ✓ | ✓ |
| elasticache-parameter-group | ElastiCacheParameterGroup | ✓ | | ✓ | ✓ |
| elasticache-serverless | ElastiCacheServerless | ✓ | ✓ | ✓ | ✓ |
| elasticache-subnet-group | ElastiCacheSubnetGroup | ✓ | | ✓ | ✓ |
| elb | ELBv1 | ✓ | ✓ | ✓ | ✓ |
| elbv2 | ELBv2 | ✓ | ✓ | ✓ | ✓ |
| event-bridge | EventBridge | ✓ | ✓ | ✓ | ✓ |
| event-bridge-archive | EventBridgeArchive | ✓ | ✓ | ✓ | ✓ |
| event-bridge-rule | EventBridgeRule | ✓ | | ✓ | ✓ |
| event-bridge-schedule | EventBridgeSchedule | ✓ | ✓ | ✓ | ✓ |
| event-bridge-schedule-group | EventBridgeScheduleGroup | ✓ | ✓ | ✓ | ✓ |
| grafana | Grafana | ✓ | ✓ | ✓ | ✓ |
| guard-duty | GuardDuty | | ✓ | ✓ | ✓ |
| iam-group | IAMGroups | ✓ | ✓ | | ✓ |
| iam-instance-profile | IAMInstanceProfiles | ✓ | ✓ | ✓ | ✓ |
| iam-policy | IAMPolicies | ✓ | ✓ | ✓ | ✓ |
| iam-role | IAMRoles | ✓ | ✓ | ✓ | ✓ |
| iam-service-linked-role | IAMServiceLinkedRoles | ✓ | ✓ | | ✓ |
| iam-user | IAMUsers | ✓ | ✓ | ✓ | ✓ |
| internet-gateway | InternetGateway | ✓ | ✓ | ✓ | ✓ |
| ipam | EC2IPAM | ✓ | ✓ | ✓ | ✓ |
| ipam-byoasn | EC2IPAMByoasn | | | | ✓ |
| ipam-custom-allocation | EC2IPAMCustomAllocation | | | | ✓ |
| ipam-pool | EC2IPAMPool | ✓ | ✓ | ✓ | ✓ |
| ipam-resource-discovery | EC2IPAMResourceDiscovery | ✓ | ✓ | ✓ | ✓ |
| ipam-scope | EC2IPAMScope | ✓ | ✓ | ✓ | ✓ |
| kinesis-firehose | KinesisFirehose | ✓ | | ✓ | ✓ |
| kinesis-stream | KinesisStream | ✓ | | ✓ | ✓ |
| kms-customer-key | KMSCustomerKeys | ✓ | ✓ | ✓ | |
| lambda | LambdaFunction | ✓ | ✓ | ✓ | ✓ |
| lambda-layer | LambdaLayer | ✓ | ✓ | ✓ | ✓ |
| launch-configuration | LaunchConfiguration | ✓ | ✓ | | ✓ |
| launch-template | LaunchTemplate | ✓ | ✓ | ✓ | ✓ |
| macie-member | MacieMember | | ✓ | ✓ | ✓ |
| managed-prometheus | ManagedPrometheus | ✓ | ✓ | ✓ | ✓ |
| mq-broker | MQBroker | ✓ | ✓ | ✓ | ✓ |
| msk-cluster | MSKCluster | ✓ | ✓ | ✓ | ✓ |
| nat-gateway | NATGateway | ✓ | ✓ | ✓ | ✓ |
| network-acl | NetworkACL | ✓ | ✓ | ✓ | ✓ |
| network-firewall | NetworkFirewall | ✓ | ✓ | ✓ | |
| network-firewall-policy | NetworkFirewallPolicy | ✓ | ✓ | ✓ | |
| network-firewall-resource-policy | NetworkFirewallResourcePolicy | ✓ | | | |
| network-firewall-rule-group | NetworkFirewallRuleGroup | ✓ | ✓ | ✓ | |
| network-firewall-tls-config | NetworkFirewallTLSConfig | ✓ | ✓ | ✓ | |
| network-interface | NetworkInterface | ✓ | ✓ | ✓ | ✓ |
| oidc-provider | OIDCProvider | ✓ | ✓ | ✓ | ✓ |
| opensearch-domain | OpenSearchDomain | ✓ | ✓ | ✓ | ✓ |
| rds-cluster | DBClusters | ✓ | ✓ | ✓ | ✓ |
| rds-global-cluster | DBGlobalClusters | ✓ | | ✓ | ✓ |
| rds-global-cluster-membership | DBGlobalClusterMemberships | ✓ | | ✓ | ✓ |
| rds-instance | DBInstances | ✓ | ✓ | ✓ | ✓ |
| rds-parameter-group | RDSParameterGroup | ✓ | | ✓ | ✓ |
| rds-proxy | RDSProxy | ✓ | ✓ | ✓ | ✓ |
| rds-cluster-snapshot | RDSClusterSnapshot | ✓ | ✓ | ✓ | ✓ |
| rds-snapshot | RDSSnapshot | ✓ | ✓ | ✓ | ✓ |
| rds-subnet-group | DBSubnetGroups | ✓ | | ✓ | ✓ |
| redshift | Redshift | ✓ | ✓ | ✓ | ✓ |
| redshift-snapshot-copy-grant | RedshiftSnapshotCopyGrant | ✓ | | ✓ | ✓ |
| resource-share | ResourceShare | ✓ | ✓ | ✓ | ✓ |
| route-table | RouteTable | ✓ | ✓ | ✓ | ✓ |
| route53-cidr-collection | Route53CIDRCollection | ✓ | | | |
| route53-hosted-zone | Route53HostedZone | ✓ | | ✓ | |
| route53-traffic-policy | Route53TrafficPolicy | ✓ | | | |
| s3 | S3 | ✓ | ✓ | ✓ | ✓ |
| s3-access-point | S3AccessPoint | ✓ | | | ✓ |
| s3-multi-region-access-point | S3MultiRegionAccessPoint | ✓ | ✓ | | ✓ |
| s3-object-lambda-access-point | S3ObjectLambdaAccessPoint | ✓ | | | ✓ |
| sagemaker-endpoint | SageMakerEndpoint | ✓ | ✓ | ✓ | ✓ |
| sagemaker-endpoint-config | SageMakerEndpointConfig | ✓ | ✓ | ✓ | ✓ |
| sagemaker-notebook-instance | SageMakerNotebook | ✓ | ✓ | ✓ | ✓ |
| sagemaker-studio | SageMakerStudioDomain | ✓ | ✓ | ✓ | ✓ |
| secrets-manager | SecretsManager | ✓ | ✓ | ✓ | ✓ |
| security-group | SecurityGroup | ✓ | ✓ | ✓ | |
| security-hub | SecurityHub | | ✓ | ✓ | ✓ |
| ses-configuration-set | SESConfigurationSet | ✓ | | | ✓ |
| ses-email-template | SESEmailTemplates | ✓ | ✓ | | ✓ |
| ses-identity | SESIdentity | ✓ | | | ✓ |
| ses-receipt-filter | SESReceiptFilter | ✓ | | | ✓ |
| ses-receipt-rule-set | SESReceiptRuleSet | ✓ | ✓ | | ✓ |
| sns-topic | SNS | ✓ | ✓ | ✓ | ✓ |
| sqs | SQS | ✓ | ✓ | ✓ | ✓ |
| ssm-parameter | SSMParameter | ✓ | ✓ | ✓ | ✓ |
| transit-gateway | TransitGateway | ✓ | ✓ | ✓ | ✓ |
| transit-gateway-attachment | TransitGatewayVPCAttachment | | ✓ | ✓ | ✓ |
| transit-gateway-peering-attachment | TransitGatewayPeeringAttachment | | ✓ | ✓ | ✓ |
| transit-gateway-route-table | TransitGatewayRouteTable | | ✓ | ✓ | ✓ |
| vpc | VPC | ✓ | ✓ | ✓ | |
| vpc-lattice-service | VPCLatticeService | ✓ | ✓ | ✓ | ✓ |
| vpc-lattice-service-network | VPCLatticeServiceNetwork | ✓ | ✓ | ✓ | ✓ |
| vpc-lattice-target-group | VPCLatticeTargetGroup | ✓ | ✓ | ✓ | ✓ |
| vpc-peering-connection | VPCPeeringConnection | ✓ | ✓ | ✓ | ✓ |

## GCP Supported Resources

cloud-nuke supports inspecting and deleting the following GCP resources. The **CLI ID** column is the value you pass to `--resource-type`.

| CLI ID | Resource |
|---|---|
| `artifact-registry` | Artifact Registry Repository |
| `cloud-function` | Cloud Functions (Gen2) |
| `gcs-bucket` | Google Cloud Storage Bucket |
| `gcp-pubsub-topic` | Pub/Sub Topic |

### GCP Config Support Matrix

| Resource Type | Config Key | names_regex | time | timeout |
|---|---|---|---|---|
| artifact-registry | ArtifactRegistry | ✓ | ✓ | ✓ |
| cloud-function | CloudFunction | ✓ | ✓ | ✓ |
| gcs-bucket | GCSBucket | ✓ | ✓ | ✓ |
| gcp-pubsub-topic | GcpPubSubTopic | ✓ | ✓ | ✓ |

## IsNukable Permission Check

For certain resources, cloud-nuke can verify whether you have sufficient permissions before attempting deletion. If not, it raises `error: INSUFFICIENT_PERMISSION`.

Supported resources: AMI, EBS, DHCP Option, Egress Only Internet Gateway, Endpoints, Internet Gateway, IPAM, IPAM BYOASN, IPAM Custom Allocation, IPAM Pool, IPAM Resource Discovery, IPAM Scope, Key Pair, Network ACL, Network Interface, Subnet, VPC, Elastic IP, Launch Template, NAT Gateway, Network Firewall, Security Group, Snapshot, Transit Gateway.

> This check relies on the AWS `DryRun` feature, which is not available for all resource types.

## Nukable Error Statuses

- `error:INSUFFICIENT_PERMISSION` — You don't have enough permission to nuke the resource.
- `error:DIFFERENT_OWNER` — You are attempting to nuke a resource for which you are not the owner.

