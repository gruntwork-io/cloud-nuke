# Supported Resources

cloud-nuke supports inspecting and deleting the following AWS resources. The **CLI ID** column is the value you pass to `--resource-type`.

| CLI ID | Resource |
|---|---|
| `accessanalyzer` | IAM Access Analyzer |
| `acm` | ACM Certificate |
| `acmpca` | ACM Private CA |
| `ami` | EC2 AMI |
| `apigateway` | API Gateway (v1) |
| `apigatewayv2` | API Gateway (v2) |
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
| `data-sync-location` | DataSync Location |
| `data-sync-task` | DataSync Task |
| `dynamodb` | DynamoDB Table |
| `ebs` | EBS Volume |
| `ec2` | EC2 Instance |
| `ec2-dedicated-hosts` | EC2 Dedicated Host |
| `ec2-dhcp-option` | EC2 DHCP Option Set |
| `ec2-endpoint` | EC2 VPC Endpoint |
| `ec2-keypairs` | EC2 Key Pair |
| `ec2-placement-groups` | EC2 Placement Group |
| `ec2-subnet` | EC2 Subnet |
| `ecr` | ECR Repository |
| `ecscluster` | ECS Cluster |
| `ecsserv` | ECS Service |
| `efs` | EFS File System |
| `egress-only-internet-gateway` | Egress Only Internet Gateway |
| `eip` | Elastic IP |
| `ekscluster` | EKS Cluster |
| `elastic-beanstalk` | Elastic Beanstalk Application |
| `elasticache` | ElastiCache Cluster |
| `elasticacheParameterGroups` | ElastiCache Parameter Group |
| `elasticacheSubnetGroups` | ElastiCache Subnet Group |
| `elasticcache-serverless` | ElastiCache Serverless Cluster |
| `elb` | Classic Load Balancer |
| `elbv2` | Application/Network Load Balancer |
| `event-bridge` | EventBridge Bus |
| `event-bridge-archive` | EventBridge Archive |
| `event-bridge-rule` | EventBridge Rule |
| `event-bridge-schedule` | EventBridge Schedule |
| `event-bridge-schedule-group` | EventBridge Schedule Group |
| `grafana` | Grafana Workspace |
| `guardduty` | GuardDuty Detector |
| `iam` | IAM User |
| `iam-group` | IAM Group |
| `iam-instance-profile` | IAM Instance Profile |
| `iam-policy` | IAM Policy |
| `iam-role` | IAM Role |
| `iam-service-linked-role` | IAM Service-linked Role |
| `internet-gateway` | Internet Gateway |
| `ipam` | EC2 IPAM |
| `ipam-byoasn` | EC2 IPAM BYOASN |
| `ipam-custom-allocation` | EC2 IPAM Custom Allocation |
| `ipam-pool` | EC2 IPAM Pool |
| `ipam-resource-discovery` | EC2 IPAM Resource Discovery |
| `ipam-scope` | EC2 IPAM Scope |
| `kinesis-firehose` | Kinesis Firehose |
| `kinesis-stream` | Kinesis Stream |
| `kmscustomerkeys` | KMS Customer Managed Key |
| `lambda` | Lambda Function |
| `lambda_layer` | Lambda Layer |
| `lc` | Launch Configuration |
| `lt` | Launch Template |
| `macie-member` | Macie Member Account |
| `managed-prometheus` | Managed Prometheus Workspace |
| `msk-cluster` | MSK Cluster |
| `nat-gateway` | NAT Gateway |
| `network-acl` | Network ACL |
| `network-firewall` | Network Firewall |
| `network-firewall-policy` | Network Firewall Policy |
| `network-firewall-resource-policy` | Network Firewall Resource Policy |
| `network-firewall-rule-group` | Network Firewall Rule Group |
| `network-firewall-tls-config` | Network Firewall TLS Config |
| `network-interface` | Network Interface |
| `oidcprovider` | OIDC Provider |
| `opensearchdomain` | OpenSearch Domain |
| `rds` | RDS DB Instance (incl. Neptune, DocumentDB) |
| `rds-cluster` | RDS DB Cluster |
| `rds-global-cluster` | RDS Global Cluster |
| `rds-global-cluster-membership` | RDS Global Cluster Membership |
| `rds-parameter-group` | RDS Parameter Group |
| `rds-proxy` | RDS Proxy |
| `rds-snapshot` | RDS Snapshot |
| `rds-subnet-group` | RDS Subnet Group |
| `redshift` | Redshift Cluster |
| `redshift-snapshot-copy-grant` | Redshift Snapshot Copy Grant |
| `route53-cidr-collection` | Route53 CIDR Collection |
| `route53-hosted-zone` | Route53 Hosted Zone |
| `route53-traffic-policy` | Route53 Traffic Policy |
| `s3` | S3 Bucket |
| `s3-ap` | S3 Access Point |
| `s3-mrap` | S3 Multi Region Access Point |
| `s3-olap` | S3 Object Lambda Access Point |
| `sagemaker-endpoint` | SageMaker Endpoint |
| `sagemaker-endpoint-config` | SageMaker Endpoint Configuration |
| `sagemaker-notebook-smni` | SageMaker Notebook Instance |
| `sagemaker-studio` | SageMaker Studio Domain |
| `secretsmanager` | Secrets Manager Secret |
| `security-group` | Security Group |
| `security-hub` | Security Hub |
| `ses-configuration-set` | SES Configuration Set |
| `ses-email-template` | SES Email Template |
| `ses-identity` | SES Identity |
| `ses-receipt-filter` | SES Receipt Filter |
| `ses-receipt-rule-set` | SES Receipt Rule Set |
| `snap` | EBS Snapshot |
| `snstopic` | SNS Topic |
| `sqs` | SQS Queue |
| `transit-gateway` | Transit Gateway |
| `transit-gateway-attachment` | Transit Gateway VPC Attachment |
| `transit-gateway-peering-attachment` | Transit Gateway Peering Attachment |
| `transit-gateway-route-table` | Transit Gateway Route Table |
| `vpc` | VPC |
| `vpc-lattice-service` | VPC Lattice Service |
| `vpc-lattice-service-network` | VPC Lattice Service Network |
| `vpc-lattice-target-group` | VPC Lattice Target Group |

> **WARNING:** The RDS APIs also interact with Neptune and DocumentDB resources. Running `cloud-nuke aws --resource-type rds` without a config file will remove any Neptune and DocumentDB resources in the account.

> **NOTE:** Resources created by AWS Backup are managed by AWS Backup and cannot be deleted through standard API calls. These resources are tagged by AWS Backup and are automatically filtered out by cloud-nuke.

## Config Support Matrix

This table shows which filtering features are supported for each resource type in the [config file](configuration.md).

| Resource Type | Config Key | names_regex | time | tags | timeout |
|---|---|---|---|---|---|
| acm | ACM | ✓ | ✓ | | ✓ |
| acmpca | ACMPCA | | ✓ | | ✓ |
| ami | AMI | ✓ | ✓ | | ✓ |
| apigateway | APIGateway | ✓ | ✓ | | ✓ |
| apigatewayv2 | APIGatewayV2 | ✓ | ✓ | | ✓ |
| accessanalyzer | AccessAnalyzer | ✓ | ✓ | | ✓ |
| asg | AutoScalingGroup | ✓ | ✓ | ✓ | ✓ |
| app-runner-service | AppRunnerService | ✓ | ✓ | | ✓ |
| backup-vault | BackupVault | ✓ | ✓ | | ✓ |
| cloudwatch-alarm | CloudWatchAlarm | ✓ | ✓ | | ✓ |
| cloudwatch-dashboard | CloudWatchDashboard | ✓ | ✓ | | ✓ |
| cloudwatch-loggroup | CloudWatchLogGroup | ✓ | ✓ | | ✓ |
| cloudtrail | CloudtrailTrail | ✓ | | ✓ | ✓ |
| cloudmap-namespace | CloudMapNamespace | ✓ | ✓ | ✓ | ✓ |
| cloudmap-service | CloudMapService | ✓ | ✓ | ✓ | ✓ |
| codedeploy-application | CodeDeployApplications | ✓ | ✓ | | ✓ |
| config-recorders | ConfigServiceRecorder | ✓ | | | ✓ |
| config-rules | ConfigServiceRule | ✓ | | | ✓ |
| data-sync-location | DataSyncLocation | | | | ✓ |
| data-sync-task | DataSyncTask | ✓ | | | ✓ |
| dynamodb | DynamoDB | ✓ | ✓ | | ✓ |
| ebs | EBSVolume | ✓ | ✓ | ✓ | ✓ |
| elastic-beanstalk | ElasticBeanstalk | ✓ | ✓ | | ✓ |
| ec2 | EC2 | ✓ | ✓ | ✓ | ✓ |
| ec2-dedicated-hosts | EC2DedicatedHosts | ✓ | ✓ | | ✓ |
| ec2-dhcp-option | EC2DhcpOption | | | | ✓ |
| ec2-keypairs | EC2KeyPairs | ✓ | ✓ | ✓ | ✓ |
| ipam | EC2IPAM | ✓ | ✓ | ✓ | ✓ |
| ipam-pool | EC2IPAMPool | ✓ | ✓ | ✓ | ✓ |
| ipam-resource-discovery | EC2IPAMResourceDiscovery | ✓ | ✓ | ✓ | ✓ |
| ipam-scope | EC2IPAMScope | ✓ | ✓ | ✓ | ✓ |
| ec2-placement-groups | EC2PlacementGroups | ✓ | ✓ | ✓ | ✓ |
| ec2-subnet | EC2Subnet | ✓ | ✓ | ✓ | |
| ec2-endpoint | EC2Endpoint | ✓ | ✓ | ✓ | ✓ |
| ecr | ECRRepository | ✓ | ✓ | | ✓ |
| ecscluster | ECSCluster | ✓ | ✓ | ✓ | ✓ |
| ecsserv | ECSService | ✓ | ✓ | ✓ | ✓ |
| ekscluster | EKSCluster | ✓ | ✓ | ✓ | ✓ |
| elb | ELBv1 | ✓ | ✓ | | ✓ |
| elbv2 | ELBv2 | ✓ | ✓ | | ✓ |
| efs | ElasticFileSystem | ✓ | ✓ | | ✓ |
| eip | ElasticIP | ✓ | ✓ | ✓ | ✓ |
| elasticache | Elasticache | ✓ | ✓ | | ✓ |
| elasticcache-serverless | ElasticCacheServerless | ✓ | ✓ | | ✓ |
| elasticacheparametergroups | ElasticacheParameterGroups | ✓ | | | ✓ |
| elasticachesubnetgroups | ElasticacheSubnetGroups | ✓ | | | ✓ |
| event-bridge | EventBridge | ✓ | ✓ | | ✓ |
| event-bridge-archive | EventBridgeArchive | ✓ | ✓ | | ✓ |
| event-bridge-rule | EventBridgeRule | ✓ | | | ✓ |
| event-bridge-schedule | EventBridgeSchedule | ✓ | ✓ | | ✓ |
| event-bridge-schedule-group | EventBridgeScheduleGroup | ✓ | ✓ | | ✓ |
| grafana | Grafana | ✓ | ✓ | ✓ | ✓ |
| guardduty | GuardDuty | | ✓ | | ✓ |
| iam-group | IAMGroups | ✓ | ✓ | | ✓ |
| iam-policy | IAMPolicies | ✓ | ✓ | ✓ | ✓ |
| iam-role | IAMRoles | ✓ | ✓ | ✓ | ✓ |
| iam-service-linked-role | IAMServiceLinkedRoles | ✓ | ✓ | | ✓ |
| iam | IAMUsers | ✓ | ✓ | ✓ | ✓ |
| internet-gateway | InternetGateway | ✓ | ✓ | ✓ | ✓ |
| egress-only-internet-gateway | EgressOnlyInternetGateway | ✓ | ✓ | ✓ | ✓ |
| kmscustomerkeys | KMSCustomerKeys | ✓ | ✓ | | |
| kinesis-stream | KinesisStream | ✓ | | | ✓ |
| kinesis-firehose | KinesisFirehose | ✓ | | | ✓ |
| lambda | LambdaFunction | ✓ | ✓ | ✓ | ✓ |
| lc | LaunchConfiguration | ✓ | ✓ | | ✓ |
| lt | LaunchTemplate | ✓ | ✓ | ✓ | ✓ |
| macie-member | MacieMember | | ✓ | | ✓ |
| msk-cluster | MSKCluster | ✓ | ✓ | | ✓ |
| managed-prometheus | ManagedPrometheus | ✓ | ✓ | ✓ | ✓ |
| nat-gateway | NatGateway | ✓ | ✓ | ✓ | ✓ |
| network-acl | NetworkACL | ✓ | ✓ | ✓ | ✓ |
| network-interface | NetworkInterface | ✓ | ✓ | ✓ | ✓ |
| oidcprovider | OIDCProvider | ✓ | ✓ | | ✓ |
| opensearchdomain | OpenSearchDomain | ✓ | ✓ | | ✓ |
| redshift | Redshift | ✓ | ✓ | | ✓ |
| rds-cluster | DBClusters | ✓ | ✓ | ✓ | ✓ |
| rds | DBInstances | ✓ | ✓ | ✓ | ✓ |
| rds-parameter-group | RdsParameterGroup | ✓ | | | ✓ |
| rds-subnet-group | DBSubnetGroups | ✓ | | | ✓ |
| rds-proxy | RDSProxy | ✓ | ✓ | | ✓ |
| s3 | s3 | ✓ | ✓ | ✓ | ✓ |
| s3-ap | s3AccessPoint | ✓ | | | ✓ |
| s3-olap | S3ObjectLambdaAccessPoint | ✓ | | | ✓ |
| s3-mrap | S3MultiRegionAccessPoint | ✓ | ✓ | | ✓ |
| security-group | SecurityGroup | ✓ | ✓ | ✓ | |
| ses-configuration-set | SesConfigurationset | ✓ | | | ✓ |
| ses-email-template | SesEmailTemplates | ✓ | ✓ | | ✓ |
| ses-identity | SesIdentity | ✓ | | | ✓ |
| ses-receipt-rule-set | SesReceiptRuleSet | ✓ | ✓ | | ✓ |
| ses-receipt-filter | SesReceiptFilter | ✓ | | | ✓ |
| snstopic | SNS | ✓ | ✓ | | ✓ |
| sqs | SQS | ✓ | ✓ | | ✓ |
| sagemaker-notebook-smni | SageMakerNotebook | ✓ | ✓ | | ✓ |
| sagemaker-endpoint | SageMakerEndpoint | ✓ | ✓ | ✓ | ✓ |
| sagemaker-studio | SageMakerStudioDomain | | | | ✓ |
| secretsmanager | SecretsManager | ✓ | ✓ | ✓ | ✓ |
| security-hub | SecurityHub | | ✓ | | ✓ |
| snap | Snapshots | | ✓ | ✓ | ✓ |
| transit-gateway | TransitGateway | | ✓ | | ✓ |
| transit-gateway-route-table | TransitGatewayRouteTable | | ✓ | | ✓ |
| transit-gateway-attachment | TransitGatewaysVpcAttachment | | ✓ | | ✓ |
| vpc | VPC | ✓ | ✓ | ✓ | |
| route53-hosted-zone | Route53HostedZone | ✓ | | | |
| route53-cidr-collection | Route53CIDRCollection | ✓ | | | |
| route53-traffic-policy | Route53TrafficPolicy | ✓ | | | |
| network-firewall | NetworkFirewall | ✓ | ✓ | ✓ | |
| network-firewall-policy | NetworkFirewallPolicy | ✓ | ✓ | ✓ | |
| network-firewall-rule-group | NetworkFirewallRuleGroup | ✓ | ✓ | ✓ | |
| network-firewall-tls-config | NetworkFirewallTLSConfig | ✓ | ✓ | ✓ | |
| network-firewall-resource-policy | NetworkFirewallResourcePolicy | ✓ | | | |
| vpc-lattice-service | VPCLatticeService | ✓ | ✓ | | ✓ |
| vpc-lattice-service-network | VPCLatticeServiceNetwork | ✓ | ✓ | | ✓ |
| vpc-lattice-target-group | VPCLatticeTargetGroup | ✓ | ✓ | | ✓ |
| cloudfront-distribution | CloudfrontDistribution | | | | ✓ |
| cloudformation-stack | CloudFormationStack | ✓ | ✓ | ✓ | ✓ |
| lambda_layer | LambdaLayer | ✓ | ✓ | | ✓ |
| rds-global-cluster | DBGlobalClusters | ✓ | | | ✓ |
| rds-global-cluster-membership | DBGlobalClusterMemberships | ✓ | | | ✓ |
| rds-snapshot | RdsSnapshot | ✓ | ✓ | ✓ | ✓ |
| redshift-snapshot-copy-grant | RedshiftSnapshotCopyGrant | ✓ | | | ✓ |
| sagemaker-endpoint-config | SageMakerEndpointConfig | ✓ | ✓ | ✓ | ✓ |
| iam-instance-profile | IAMInstanceProfiles | ✓ | ✓ | ✓ | ✓ |
| transit-gateway-peering-attachment | TransitGatewayPeeringAttachment | | ✓ | | ✓ |
| ipam-byoasn | EC2IPAMByoasn | | | | ✓ |
| ipam-custom-allocation | EC2IPAMCustomAllocation | | | | ✓ |

## IsNukable Permission Check

For certain resources, cloud-nuke can verify whether you have sufficient permissions before attempting deletion. If not, it raises `error: INSUFFICIENT_PERMISSION`.

Supported resources: AMI, EBS, DHCP Option, Egress Only Internet Gateway, Endpoints, Internet Gateway, IPAM, IPAM BYOASN, IPAM Custom Allocation, IPAM Pool, IPAM Resource Discovery, IPAM Scope, Key Pair, Network ACL, Network Interface, Subnet, VPC, Elastic IP, Launch Template, NAT Gateway, Network Firewall, Security Group, Snapshot, Transit Gateway.

> This check relies on the AWS `DryRun` feature, which is not available for all resource types.

## Nukable Error Statuses

- `error:INSUFFICIENT_PERMISSION` — You don't have enough permission to nuke the resource.
- `error:DIFFERENT_OWNER` — You are attempting to nuke a resource for which you are not the owner.
