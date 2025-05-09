[![Maintained by Gruntwork.io](https://img.shields.io/badge/maintained%20by-gruntwork.io-%235849a6.svg)](https://gruntwork.io/?ref=repo_cloud_nuke)

# cloud-nuke

This repo contains a CLI tool to delete all resources . cloud-nuke was created for situations when you might have an
account you use for testing and need to clean up leftover resources so you're not charged for them. Also great for
cleaning out accounts with redundant resources. Also great for removing unnecessary defaults like default VPCs and
permissive ingress/egress rules in default security groups.

In addition, cloud-nuke offers non-destructive inspecting functionality that can either be called via the command-line
interface, or consumed as library methods, for scripting purposes.

The currently supported functionality includes:

## AWS

Cloud-nuke supports 🔎 inspecting and 🔥💀 deleting the following AWS resources. Each resource has a name (as identified in AWS), a type (as used in the code), and a config key (as used in YAML configuration):

| Resource Name | Resource Type | Config Key | Name Regex | Time Filter | Tags | Timeout |
|--------------|---------------|------------|------------|-------------|------|---------|
| access-analyzer | AccessAnalyzer | AccessAnalyzer | ✅ | ✅ | ❌ | ✅ |
| acmpca | ACMPCA | ACMPCA | ❌ | ✅ | ❌ | ✅ |
| ami | AMI | AMI | ✅ | ✅ | ❌ | ✅ |
| apigateway | APIGateway | APIGateway | ✅ | ✅ | ❌ | ✅ |
| apigatewayv2 | ApiGatewayV2 | APIGatewayV2 | ✅ | ✅ | ❌ | ✅ |
| app-runner-service | AppRunnerService | AppRunnerService | ✅ | ✅ | ❌ | ✅ |
| backup-vault | BackupVault | BackupVault | ✅ | ✅ | ❌ | ✅ |
| cloudwatch-alarm | CloudWatchAlarms | CloudWatchAlarm | ✅ | ✅ | ❌ | ✅ |
| cloudwatch-dashboard | CloudWatchDashboards | CloudWatchDashboard | ✅ | ✅ | ❌ | ✅ |
| cloudwatch-log-group | CloudWatchLogGroup | CloudWatchLogGroup | ✅ | ✅ | ❌ | ✅ |
| cloudtrail-trail | CloudtrailTrail | CloudtrailTrail | ✅ | ❌ | ❌ | ✅ |
| cloudfront-distribution | CloudfrontDistribution | CloudfrontDistribution | ✅ | ✅ | ❌ | ✅ |
| codedeploy-application | CodeDeployApplications | CodeDeployApplications | ✅ | ✅ | ❌ | ✅ |
| config-recorder | ConfigServiceRecorder | ConfigServiceRecorder | ✅ | ❌ | ❌ | ✅ |
| config-rule | ConfigServiceRule | ConfigServiceRule | ✅ | ❌ | ❌ | ✅ |
| data-sync-location | DataSyncLocation | DataSyncLocation | ❌ | ❌ | ❌ | ✅ |
| data-sync-task | DataSyncTask | DataSyncTask | ✅ | ❌ | ❌ | ✅ |
| dynamodb | DynamoDB | DynamoDB | ✅ | ✅ | ❌ | ✅ |
| ebs-volume | EBSVolume | EBSVolume | ✅ | ✅ | ✅ | ✅ |
| elastic-beanstalk | ElasticBeanstalk | ElasticBeanstalk | ✅ | ✅ | ❌ | ✅ |
| ec2 | EC2 | EC2 | ✅ | ✅ | ✅ | ✅ |
| ec2-dedicated-host | EC2DedicatedHosts | EC2DedicatedHosts | ✅ | ✅ | ❌ | ✅ |
| ec2-dhcp-option | EC2DhcpOption | EC2DhcpOption | ❌ | ❌ | ❌ | ✅ |
| ec2-keypair | EC2KeyPairs | EC2KeyPairs | ✅ | ✅ | ✅ | ✅ |
| ec2-ipam | EC2IPAM | EC2IPAM | ✅ | ✅ | ✅ | ✅ |
| ec2-ipam-pool | EC2IPAMPool | EC2IPAMPool | ✅ | ✅ | ✅ | ✅ |
| ec2-ipam-resource-discovery | EC2IPAMResourceDiscovery | EC2IPAMResourceDiscovery | ✅ | ✅ | ✅ | ✅ |
| ec2-ipam-scope | EC2IPAMScope | EC2IPAMScope | ✅ | ✅ | ✅ | ✅ |
| ec2-placement-group | EC2PlacementGroups | EC2PlacementGroups | ✅ | ✅ | ✅ | ✅ |
| ec2-subnet | EC2Subnet | EC2Subnet | ✅ | ✅ | ✅ | ❌ |
| ec2-endpoint | EC2Endpoint | EC2Endpoint | ✅ | ✅ | ✅ | ✅ |
| ecr-repository | ECRRepository | ECRRepository | ✅ | ✅ | ❌ | ✅ |
| ecs-cluster | ECSCluster | ECSCluster | ✅ | ❌ | ❌ | ✅ |
| ecs-service | ECSService | ECSService | ✅ | ✅ | ❌ | ✅ |
| ekscluster | EKSClusters | EKSCluster | ✅ | ✅ | ✅ | ✅ |
| elastic-beanstalk | ElasticBeanstalk | ElasticBeanstalk | ✅ | ✅ | ❌ | ✅ |
| efs | ElasticFileSystem | ElasticFileSystem | ✅ | ✅ | ❌ | ✅ |
| elastic-ip | ElasticIP | ElasticIP | ✅ | ✅ | ✅ | ✅ |
| elasticache | Elasticaches | Elasticache | ✅ | ✅ | ❌ | ✅ |
| elasticache-parameter-group | ElasticacheParameterGroups | ElasticacheParameterGroups | ✅ | ❌ | ❌ | ✅ |
| elasticache-serverless | ElasticCacheServerless | ElasticCacheServerless | ✅ | ✅ | ❌ | ✅ |
| elasticache-subnet-group | ElasticacheSubnetGroups | ElasticacheSubnetGroups | ✅ | ❌ | ❌ | ✅ |
| elb-v1 | ELBv1 | ELBv1 | ✅ | ✅ | ❌ | ✅ |
| elb-v2 | ELBv2 | ELBv2 | ✅ | ✅ | ❌ | ✅ |
| event-bridge | EventBridge | EventBridge | ✅ | ✅ | ❌ | ✅ |
| event-bridge-archive | EventBridgeArchive | EventBridgeArchive | ✅ | ✅ | ❌ | ✅ |
| event-bridge-rule | EventBridgeRule | EventBridgeRule | ✅ | ❌ | ❌ | ✅ |
| event-bridge-schedule | EventBridgeSchedule | EventBridgeSchedule | ✅ | ✅ | ❌ | ✅ |
| event-bridge-schedule-group | EventBridgeScheduleGroup | EventBridgeScheduleGroup | ✅ | ✅ | ❌ | ✅ |
| grafana | Grafana | Grafana | ✅ | ✅ | ✅ | ✅ |
| guardduty | GuardDuty | GuardDuty | ❌ | ✅ | ❌ | ✅ |
| iam-group | IAMGroups | IAMGroups | ✅ | ✅ | ❌ | ✅ |
| iam-policy | IAMPolicies | IAMPolicies | ✅ | ✅ | ❌ | ✅ |
| iam-role | IAMRoles | IAMRoles | ✅ | ✅ | ❌ | ✅ |
| iam-service-linked-role | IAMServiceLinkedRoles | IAMServiceLinkedRoles | ✅ | ✅ | ❌ | ✅ |
| iam-user | IAMUsers | IAMUsers | ✅ | ✅ | ✅ | ✅ |
| iam-instance-profile | IAMInstanceProfiles | IAMInstanceProfiles | ✅ | ✅ | ❌ | ✅ |
| internet-gateway | InternetGateway | InternetGateway | ✅ | ✅ | ✅ | ✅ |
| egress-only-internet-gateway | EgressOnlyInternetGateway | EgressOnlyInternetGateway | ✅ | ✅ | ✅ | ✅ |
| kms-customer-key | KMSCustomerKeys | KMSCustomerKeys | ✅ | ✅ | ❌ | ❌ |
| kinesis-stream | KinesisStream | KinesisStream | ✅ | ❌ | ❌ | ✅ |
| kinesis-firehose | KinesisFirehose | KinesisFirehose | ✅ | ❌ | ❌ | ✅ |
| lambda | LambdaFunctions | LambdaFunction | ✅ | ✅ | ❌ | ✅ |
| launch-configuration | LaunchConfiguration | LaunchConfiguration | ✅ | ✅ | ❌ | ✅ |
| launch-template | LaunchTemplate | LaunchTemplate | ✅ | ✅ | ❌ | ✅ |
| macie-member | MacieMember | MacieMember | ❌ | ✅ | ❌ | ✅ |
| msk-cluster | MSKCluster | MSKCluster | ✅ | ✅ | ❌ | ✅ |
| managed-prometheus | ManagedPrometheus | ManagedPrometheus | ✅ | ✅ | ✅ | ✅ |
| nat-gateway | NatGateway | NatGateway | ✅ | ✅ | ✅ | ✅ |
| network-acl | NetworkACL | NetworkACL | ✅ | ✅ | ✅ | ✅ |
| network-interface | NetworkInterface | NetworkInterface | ✅ | ✅ | ✅ | ✅ |
| network-firewall | NetworkFirewall | NetworkFirewall | ✅ | ✅ | ✅ | ❌ |
| network-firewall-policy | NetworkFirewallPolicy | NetworkFirewallPolicy | ✅ | ✅ | ✅ | ❌ |
| network-firewall-rule-group | NetworkFirewallRuleGroup | NetworkFirewallRuleGroup | ✅ | ✅ | ✅ | ❌ |
| network-firewall-tls-config | NetworkFirewallTLSConfig | NetworkFirewallTLSConfig | ✅ | ✅ | ✅ | ❌ |
| network-firewall-resource-policy | NetworkFirewallResourcePolicy | NetworkFirewallResourcePolicy | ✅ | ❌ | ❌ | ❌ |
| oidc-provider | OIDCProvider | OIDCProvider | ✅ | ✅ | ❌ | ✅ |
| opensearch-domain | OpenSearchDomain | OpenSearchDomain | ✅ | ✅ | ❌ | ✅ |
| rds-cluster | DBClusters | DBClusters | ✅ | ✅ | ✅ | ✅ |
| rds-instance | DBInstances | DBInstances | ✅ | ✅ | ✅ | ✅ |
| rds-parameter-group | RdsParameterGroup | RdsParameterGroup | ✅ | ❌ | ❌ | ✅ |
| rds-subnet-group | DBSubnetGroups | DBSubnetGroups | ✅ | ❌ | ❌ | ✅ |
| rds-proxy | RDSProxy | RDSProxy | ✅ | ✅ | ❌ | ✅ |
| redshift | RedshiftClusters | Redshift | ✅ | ✅ | ❌ | ✅ |
| route53-hosted-zone | Route53HostedZone | Route53HostedZone | ✅ | ❌ | ❌ | ❌ |
| route53-cidr-collection | Route53CidrCollection | Route53CIDRCollection | ✅ | ❌ | ❌ | ❌ |
| route53-traffic-policy | Route53TrafficPolicy | Route53TrafficPolicy | ✅ | ❌ | ❌ | ❌ |
| s3 | S3Buckets | s3 | ✅ | ✅ | ✅ | ✅ |
| s3-access-point | s3AccessPoint | s3AccessPoint | ✅ | ❌ | ❌ | ✅ |
| s3-object-lambda-access-point | S3ObjectLambdaAccessPoint | S3ObjectLambdaAccessPoint | ✅ | ❌ | ❌ | ✅ |
| s3-multi-region-access-point | S3MultiRegionAccessPoint | S3MultiRegionAccessPoint | ✅ | ✅ | ❌ | ✅ |
| sagemaker-notebook-smni | SageMakerNotebookInstances | SageMakerNotebook | ✅ | ✅ | ❌ | ✅ |
| sagemaker-endpoint | SageMakerEndpoint | SageMakerEndpoint | ✅ | ✅ | ✅ | ✅ |
| sagemaker-endpoint-config | SageMakerEndpointConfig | SageMakerEndpointConfig | ✅ | ✅ | ✅ | ✅ |
| sagemaker-studio-domain | SageMakerStudioDomain | SageMakerStudioDomain | ❌ | ❌ | ❌ | ✅ |
| secretsmanager | SecretsManagerSecrets | SecretsManager | ✅ | ✅ | ❌ | ✅ |
| security-group | SecurityGroup | SecurityGroup | ✅ | ✅ | ✅ | ❌ |
| security-hub | SecurityHub | SecurityHub | ❌ | ✅ | ❌ | ✅ |
| ses-configuration-set | SesConfigurationset | SesConfigurationset | ✅ | ❌ | ❌ | ✅ |
| ses-email-template | SesEmailTemplates | SesEmailTemplates | ✅ | ✅ | ❌ | ✅ |
| ses-identity | SesIdentity | SesIdentity | ✅ | ❌ | ❌ | ✅ |
| ses-receipt-rule-set | SesReceiptRuleSet | SesReceiptRuleSet | ✅ | ❌ | ❌ | ✅ |
| ses-receipt-filter | SesReceiptFilter | SesReceiptFilter | ✅ | ❌ | ❌ | ✅ |
| sns-topic | SNS | SNS | ✅ | ✅ | ❌ | ✅ |
| sqs-queue | SQS | SQS | ✅ | ✅ | ❌ | ✅ |
| snapshot | Snapshots | Snapshots | ❌ | ✅ | ✅ | ✅ |
| transit-gateway | TransitGateway | TransitGateway | ❌ | ✅ | ❌ | ✅ |
| transit-gateway-route-table | TransitGatewayRouteTable | TransitGatewayRouteTable | ❌ | ✅ | ❌ | ✅ |
| transit-gateway-vpc-attachment | TransitGatewaysVpcAttachment | TransitGatewaysVpcAttachment | ❌ | ✅ | ❌ | ✅ |
| vpc | VPC | VPC | ✅ | ✅ | ✅ | ❌ |
| vpc-lattice-service | VPCLatticeService | VPCLatticeService | ✅ | ✅ | ❌ | ✅ |
| vpc-lattice-service-network | VPCLatticeServiceNetwork | VPCLatticeServiceNetwork | ✅ | ✅ | ❌ | ✅ |
| vpc-lattice-target-group | VPCLatticeTargetGroup | VPCLatticeTargetGroup | ✅ | ✅ | ❌ | ✅ |

### Resource Filtering Capabilities

cloud-nuke provides several ways to selectively target resources for deletion. Each resource type supports different filtering capabilities that can be configured either through command-line flags or a configuration file.

#### Basic Filtering Methods

1. **Name Regex Filter**
   - Filter resources by their name using regular expressions
   - Can be used to include or exclude resources based on name patterns
   - Example:
     ```yaml
     s3:
       include:
         names_regex:
           - ^alb-.*-access-logs$
           - .*-prod-alb-.*
       exclude:
         names_regex:
           - public
           - prod
     ```

2. **Time Filter**
   - Filter resources based on their creation time
   - Can specify resources created before or after a specific date
   - Example:
     ```yaml
     s3:
       include:
         time_after: '2020-01-01T00:00:00Z'
         time_before: '2021-01-01T00:00:00Z'
     ```

3. **Tag Filter**
   - Filter resources based on their tags using the `tags` map syntax
   - Supports multiple tags with regular expression values
   - Resources matching any of the specified tag patterns will be excluded (logical OR)
   - Example:
     ```yaml
     ec2:
       exclude:
         tags:
           Schedule: "^schedule-.*"
           Environment: "^(Prod|Dev)$"
     ```

#### Additional Filtering Features

1. **Resource Protection**
   - Resources can be protected from deletion using:
     - The `cloud-nuke-excluded=true` tag
     - The `cloud-nuke-after` tag with an ISO 8601 timestamp
     - Resource-specific protection mechanisms (e.g., deletion protection for RDS instances)

2. **Timeout Configuration**
   - Set individual timeout options for specific resources
   - Example:
     ```yaml
     s3:
       timeout: 10m  # Timeout after 10 minutes
     ```

3. **Region Filtering**
   - Filter resources by AWS regions using `--region` or `--exclude-region` flags
   - Global resources are managed in the global region
   - For GovCloud accounts, set `CLOUD_NUKE_AWS_GLOBAL_REGION` to manage global resources

#### Filtering Behavior

- Filtering is **commutative**, meaning you get the same result whether you apply include filters before or after exclude filters
- CLI options override config file options
- Tag-based filtering only supports excluding resources, not including them
- The older single-tag syntax using `tag` and `tag_value` fields is deprecated

> **Note:** Some resources may not support all filtering capabilities due to AWS API limitations. The eligibility check for nukability relies on the AWS `DryRun` feature, which is not available for all delete APIs.

### Resource Deletion Order

Resources are deleted in a specific order to handle dependencies correctly. For example:
- EC2 instances are deleted before VPCs
- Transit Gateway VPC attachments are deleted before VPCs
- DHCP options are deleted after VPCs

### Global vs Regional Resources

Some AWS resources are global (not tied to a specific region) while others are regional. When using cloud-nuke:
- Global resources are managed in the global region
- Regional resources are managed in each specified region
- For GovCloud accounts, you must set `CLOUD_NUKE_AWS_GLOBAL_REGION` to manage global resources


> **WARNING:** The RDS APIs also interact with neptune and document db resources.
> Running `cloud-nuke aws --resource-type rds` without a config file will remove any neptune and document db resources
> in
> the account.

> **NOTE: AWS Backup Resource:** Resources (such as AMIs) created by AWS Backup, while owned by your AWS account, are
> managed specifically by AWS Backup and cannot be deleted through standard APIs calls for that resource. These
> resources
> are tagged by AWS Backup and are filtered out so that `cloud-nuke` does not fail when trying to delete resources it
> cannot delete.

### BEWARE!

When executed as `cloud-nuke aws`, this tool is **HIGHLY DESTRUCTIVE** and deletes all resources! This mode should never
be used in a production environment!

When executed as `cloud-nuke defaults-aws`, this tool deletes all DEFAULT VPCs and the default ingress/egress rule for
all default security groups. This should be used in production environments **WITH CAUTION**.

## Telemetry

As of version `v0.29.0` cloud-nuke sends telemetry back to Gruntwork to help us better prioritize bug fixes and feature
improvements. The following metrics are included:

- Command and Arguments
- Version Number
- Timestamps
- Resource Types
- Resource Counts
- A randomly generated Run ID
- AWS Account ID

We never collect

- IP Addresses
- Resource Names

Telemetry can be disabled entirely by setting the `DISABLE_TELEMETRY` environment variable on the command line.

As an open source tool, you can see the exact statistics being collected by searching the code for
`telemetry.TrackEvent(...)`

## Install

### Download from releases page

1. Download the latest binary for your OS on the [releases page](https://github.com/gruntwork-io/cloud-nuke/releases).
2. Move the binary to a folder on your `PATH`. E.g.: `mv cloud-nuke_darwin_amd64 /usr/local/bin/cloud-nuke`.
3. Add execute permissions to the binary. E.g.: `chmod u+x /usr/local/bin/cloud-nuke`.
4. Test it installed correctly: `cloud-nuke --help`.

### Install via package manager

Note that package managers are third party. The third party cloud-nuke packages may not be updated with the latest
version, but are often close. Please check your version against the latest available on
the [releases page](https://github.com/gruntwork-io/cloud-nuke/releases). If you want the latest version, the
recommended installation option is
to [download from the releases page](https://github.com/gruntwork-io/cloud-nuke/releases).

- **macOS:** You can install cloud-nuke using [Homebrew](https://brew.sh/): `brew install cloud-nuke`.

- **Linux:** Most Linux users can use [Homebrew](https://docs.brew.sh/Homebrew-on-Linux): `brew install cloud-nuke`.

- **Windows:** You can install cloud-nuke using [winget](https://learn.microsoft.com/en-us/windows/package-manager/winget/): `winget install cloud-nuke`

## Usage

Simply running `cloud-nuke aws` will start the process of cleaning up your cloud account. You'll be shown a list of
resources that'll be deleted as well as a prompt to confirm before any deletion actually takes place.

In AWS, to delete only the default resources, run `cloud-nuke defaults-aws`. This will remove the default VPCs in each
region, and will also revoke the ingress and egress rules associated with the default security group in each VPC. Note
that the default security group itself is unable to be deleted.

### Nuke or inspect resources using AWS Profile

When using `cloud-nuke aws`, or `cloud-nuke inspect-aws`, you can pass in the `AWS_PROFILE` env variable to target
resources in certain regions for a specific AWS account. For example the following command will nuke resources only
in `ap-south-1` and `ap-south-2` regions in the `gruntwork-dev` AWS account:

```shell
AWS_PROFILE=gruntwork-dev cloud-nuke aws --region ap-south-1 --region ap-south-2
```

Similarly, the following command will inspect resources only in `us-east-1`

```shell
AWS_PROFILE=gruntwork-dev cloud-nuke inspect-aws --region us-east-1
```

### Nuke or inspect resources in certain regions

When using `cloud-nuke aws`, or `cloud-nuke inspect-aws`, you can use the `--region` flag to target resources in certain
regions. For example the following command will nuke resources only in `ap-south-1` and `ap-south-2` regions:

```shell
cloud-nuke aws --region ap-south-1 --region ap-south-2
```

Similarly, the following command will inspect resources only in `us-east-1`

```shell
cloud-nuke inspect-aws --region us-east-1
```

Including regions is available within:

- `cloud-nuke aws`
- `cloud-nuke defaults-aws`
- `cloud-nuke inspect-aws`

### Exclude resources in certain regions

When using `cloud-nuke aws` or `cloud-nuke inspect-aws`, you can use the `--exclude-region` flag to exclude resources in
certain regions from being deleted or inspected. For example the following command does not nuke resources
in `ap-south-1` and `ap-south-2` regions:

```shell
cloud-nuke aws --exclude-region ap-south-1 --exclude-region ap-south-2
```

Similarly, the following command will not inspect resources in the `us-west-1` region:

```shell
cloud-nuke inspect-aws --exclude-region us-west-1
```

`--region` and `--exclude-region` flags cannot be specified together i.e. they are mutually exclusive.

Excluding regions is available within:

- `cloud-nuke aws`
- `cloud-nuke defaults-aws`
- `cloud-nuke inspect-aws`

### Excluding Resources by Age

You can use the `--older-than` flag to only nuke resources that were created before a certain period, the possible
values are all valid values for [ParseDuration](https://golang.org/pkg/time/#ParseDuration) For example the following
command nukes resources that are at least one day old:

```shell
cloud-nuke aws --older-than 24h
```

Excluding resources by age is available within:

- `cloud-nuke aws`
- `cloud-nuke inspect-aws`

### List supported resource types

You can use the `--list-resource-types` flag to list resource types whose termination is currently supported:

```shell
cloud-nuke aws --list-resource-types
```

Listing supported resource types is available within:

- `cloud-nuke aws`
- `cloud-nuke inspect-aws`

### Terminate or inspect specific resource types

If you want to target specific resource types (e.g ec2, ami, etc.) instead of all the supported resources you can
do so by specifying them through the `--resource-type` flag:

```shell
cloud-nuke aws --resource-type ec2 --resource-type ami
```

will search and target only `ec2` and `ami` resources. The specified resource type should be a valid resource type
i.e. it should be present in the `--list-resource-types` output. Using `--resource-type` also speeds up search because
we are searching only for specific resource types.

Similarly, the following command will inspect only ec2 instances:

```shell
cloud-nuke inspect-aws --resource-type ec2
```

Specifying target resource types is available within:

- `cloud-nuke aws`
- `cloud-nuke inspect-aws`

### Exclude terminating specific resource types

Just like you can select which resources to terminate using `--resource-type`, you can select which resources to skip
using
`--exclude-resource-type` flag:

```shell
cloud-nuke aws --exclude-resource-type s3 --exclude-resource-type ec2
```

This will terminate all resource types other than S3 and EC2.

`--resource-type` and `--exclude-resource-type` flags cannot be specified together i.e. they are mutually exclusive.

Specifying resource types to exclude is available within:

- `cloud-nuke aws`
- `cloud-nuke inspect-aws`

### Dry run mode

If you want to check what resources are going to be targeted without actually terminating them, you can use the
`--dry-run` flag

```shell
cloud-nuke aws --resource-type ec2 --dry-run
```

Dry run mode is only available within:

- `cloud-nuke aws`

### With Timeout
If you want to set up a timeout option for resources, limiting their execution to a specified duration for nuking, use the
`--timeout` flag:

```shell
cloud-nuke aws --resource-type s3 --timeout 10m
```
This will attempt to nuke the specified resources within a 10-minute timeframe.


### Protect Resources with `cloud-nuke-after` Tag
By tagging resources with `cloud-nuke-after` and specifying a future date in ISO 8601 format (e.g., 2024-07-09T00:00:00Z), you can ensure that these resources are protected from accidental or premature deletion until the specified date. This method helps to keep important resources intact until their designated expiration date.


### Using cloud-nuke as a library

You can import cloud-nuke into other projects and use it as a library for programmatically inspecting and counting
resources.

```golang
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	nuke_aws "github.com/gruntwork-io/cloud-nuke/aws"
	nuke_config "github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/externalcreds"
)

func main() {
	// You can scan multiple regions at once, or just pass a single region for speed
	targetRegions := []string{"us-east-1", "us-west-1", "us-west-2"}
	excludeRegions := []string{}
	// You can simultaneously target multiple resource types as well
	resourceTypes := []string{"ec2", "vpc"}
	excludeResourceTypes := []string{}
	// excludeAfter is parsed identically to the --older-than flag
	excludeAfter := time.Now()
	// an optional start time- can pass null if the filter is not required
	includeAfter := time.Now().AddDate(-1, 0, 0)

	// an optional execution timeout duration
	timeout := time.Duration(10 * time.Second)

	// Any custom settings you want
	myCustomConfig := &aws.Config{}
	myCustomConfig.WithMaxRetries(3)
	myCustomConfig.WithLogLevel(aws.LogDebugWithRequestErrors)
	// Optionally, set custom credentials
	// myCustomConfig.WithCredentials()

	// Be sure to set your config prior to calling any library methods such as NewQuery
	externalcreds.Set(myCustomConfig)
	// this config can be configured to add include/exclude rule to filter the resources- for all resources pass an empty struct
	nukeConfig := nuke_config.Config{}

	// NewQuery is a convenience method for configuring parameters you want to pass to your resource search
	query, err := nuke_aws.NewQuery(
		targetRegions,
		excludeRegions,
		resourceTypes,
		excludeResourceTypes,
		&excludeAfter,
		&includeAfter,
		false,
		&timeout,
	)
	if err != nil {
		fmt.Println(err)
	}

	// GetAllResources still returns *AwsAccountResources, but this struct has been extended with several
	// convenience methods for quickly determining if resources exist in a given region
	accountResources, err := nuke_aws.GetAllResources(context.Background(), query, nukeConfig)
	if err != nil {
		fmt.Println(err)
	}
	// You can call GetRegion to examine a single region's resources
	usWest1Resources := accountResources.GetRegion("us-west-1")
	// Then interrogate them with the new methods:
	// Count the number of any resource type within the region
	countOfEc2InUsWest1 := usWest1Resources.CountOfResourceType("ec2")
	fmt.Printf("countOfEc2InUsWest1: %d\n", countOfEc2InUsWest1)
	// countOfEc2InUsWest1: 2
	fmt.Printf("usWest1Resources.ResourceTypePresent(\"ec2\"):%b\n", usWest1Resources.ResourceTypePresent("ec2"))
	// usWest1Resources.ResourceTypePresent("ec2"): true
	// Get all the resource identifiers for a given resource type
	// In this example, we're only looking for ec2 instances
	resourceIds := usWest1Resources.IdentifiersForResourceType("ec2")
	fmt.Printf("resourceIds: %s", resourceIds)
	// resourceIds:  [i-0c5d16c3ef28dda24 i-09d9739e1f4d27814]
}

```

## Config file

You can also specify which resources to terminate with more granularity via using config files. The config file is a
YAML file that specifies which resources to terminate. The top level keys of the config file are the resource types, and
the values are the rules for which resources to terminate. See [examples folder](./config/examples) for more reference.

### How to Use

Once you created your config file, you can run a command like this to nuke resources with your config file:

```shell
cloud-nuke aws --resource-type s3 --config path/to/file.yaml
```

> **CLI options override config file options**
>
> The options provided in the command line take precedence over those provided in any config file that gets passed in.
> For
> example, say you provide `--resource-type s3` in the command line, along with a config file that specifies `ec2:` at
> the
> top level but doesn't specify `s3:`. The command line argument filters the resource types to include only s3, so the
> rules in the config file for `ec2:` are ignored, and ec2 resources are not nuked. All s3 resources would be nuked.
>
> In the same vein, say you do not provide a `--resource-type` option in the command line, but you do pass in a config
> file that only lists rules for `s3:`, such as `cloud-nuke aws --config path/to/config.yaml`. In this case _all_
> resources would be nuked, but among `s3` buckets, only those matching your config file rules would be nuked.
>
> Be careful when nuking and append the `--dry-run` option if you're unsure. Even without `--dry-run`, `cloud-nuke` will
> list resources that would undergo nuking and wait for your confirmation before carrying it out.

## Log level

By default, cloud-nuke sends most output to the `Debug` level logger, to enhance legibility, since the results of every
deletion attempt will be displayed in the report that cloud-nuke prints after each run.

However, sometimes it's helpful to see all output, such as when you're debugging something.

You can set the log level by specifying the `--log-level` flag as per [logrus](https://github.com/sirupsen/logrus) log
levels:

```shell
cloud-nuke aws --log-level debug
```

OR

```shell
LOG_LEVEL=debug cloud-nuke aws
```

Default value is - `info`. Acceptable values are `debug, info, warn, error, panic, fatal, trace` as
per [logrus log level parser](https://github.com/sirupsen/logrus/blob/master/logrus.go#L25).

### Nuking only default security group rules

When deleting defaults with `cloud-nuke defaults-aws`, use the `--sg-only` flag to delete only the default
security group rules and not the default VPCs.

```shell
cloud-nuke defaults-aws --sg-only
```

## Note for nuking VPCs

When nuking VPCs cloud-nuke will attempt to remove dependency resources underneath the VPC

### Supported VPC sub-resources

- Internet Gateways
- Egress Only Internet Gateways
- Elastic Network Interfaces
- VPC Endpoints
- Subnets
- Route Tables
- Network ACLs
- Security Groups
- DHCP Option Sets (Will be dissociated from VPC, not deleted. Must be cleaned up separately)
- Elastic IPs (Supported as a separate resource that gets cleaned up first. If you are filtering what gets nuked,
  Elastic IPs may prevent VPCs from destroying.)

All other resources that get created within VPCs must be cleaned up prior to running cloud-nuke on VPC resources.

> VPC resources may not be entirely cleaned up on the first run. We believe this is caused by an eventual consistency
> error in AWS.
>
> If you see errors like `InvalidParameterValue: Network interface is currently in use.` We recommend waiting 30 minutes
> and trying again.

Happy Nuking!!!

## Credentials

### AWS

In order for the `cloud-nuke` CLI tool to access your AWS, you will need to provide your AWS credentials. You can use
one of the [standard AWS CLI credential mechanisms](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html).

#### GovCloud Configuration

When running cloud-nuke against an AWS GovCloud account, you must set the `CLOUD_NUKE_AWS_GLOBAL_REGION` environment variable to specify the global region (e.g., `us-gov-west-1`). This is required for cloud-nuke to properly discover and manage global resources like IAM users in GovCloud environments.

```shell
export CLOUD_NUKE_AWS_GLOBAL_REGION=us-gov-west-1
cloud-nuke aws
```

This environment variable is only required for GovCloud accounts and is not needed for AWS Commercial accounts. If not set when running against a GovCloud account, you may encounter errors like "the security token included in the request is invalid" when attempting to manage global resources.

Note that cloud-nuke does not use the standard `AWS_DEFAULT_REGION` environment variable for this purpose.

## Running Tests

```shell
go test -v ./...
```

## Contributing

cloud-nuke is an open source project, and contributions from the community are very welcome! Please check out the
[Contribution Guidelines](CONTRIBUTING.md) and [Developing cloud-nuke](#developing-cloud-nuke) for instructions.

## Developing cloud-nuke

### Running Locally

To run cloud-nuke locally, use the `go run` command:

```bash
go run main.go
```

### Running tests

**Note**: Many of the tests in the `aws` folder run against a real AWS account and will create and destroy actual
resources. DO NOT
hit `CTRL+C` while the tests are running, as this will prevent them from cleaning up properly. We are not responsible
for any
charges you may incur.

Before running the tests, you must configure your [AWS credentials](#credentials).

To run all the tests:

```bash
go test -v ./...
```

To run only the tests in a specific package, such as the package `aws`:

```bash
cd aws
go test -v
```

And to run a specific test, such as `TestListAMIs` in package `aws`:

```bash
cd aws
go test -v -run TestListAMIs
```

And to run a specific test, such as `TestLambdaFunction_GetAll` in package `aws/resources`:

```bash
cd aws/resources
go test -v -run TestLambdaFunction_GetAll
```

Use env-vars to opt-in to special tests, which are expensive to run:

```bash
# Run acmpca tests
TEST_ACMPCA_EXPENSIVE_ENABLE=1 go test -v ./...
```

### Formatting

Every source file in this project should be formatted with `go fmt`.

### Releasing new versions

We try to follow the release process as deifned in
our [Coding Methodology](https://www.notion.so/gruntwork/Gruntwork-Coding-Methodology-02fdcd6e4b004e818553684760bf691e#08b68ee0e19143e89523dcf483d2bf48).

#### Choosing a new release tag

If the new release contains any new resources that `cloud-nuke` will support, mark it as a minor version bump (X in
v0.X.Y)
to indicate backward incompatibilities.

This is because since version `v0.2.0` `cloud-nuke` has been configured to automatically include new resources (so you
have
to explicitly opt-out). This is inherently not backward compatible, because users with CI practices around `cloud-nuke`
would
be surprised by new resources that are suddenly being picked up for deletion! This surprise is more alarming for
resources
that are actively in use for any account, such as IAM Users.

Therefore please mark your release as backward incompatible and bump the **minor version** (`X` in `v0.X.Y`) when it
includes
support for nuking new resources, so that we provide better signals for users when we introduce a new resource.

#### To release a new version

Go to the [Releases Page](https://github.com/gruntwork-io/cloud-nuke/releases) and create a new release. The CircleCI
job for this repo has been configured to:

1. Automatically detect new tags.
1. Build binaries for every OS using that tag as a version number.
1. Upload the binaries to the release in GitHub.

See `.circleci/config.yml` for details.

## Nukable error statuses
You'll encounter any of the following statuses when attempting to nuke resources, and here's what each status means:
- `error:INSUFFICIENT_PERMISSION` : You don't have enough permission to nuke the resource.
- `error:DIFFERENT_OWNER` : You are attempting to nuke a resource for which you are not the owner.

## License

This code is released under the MIT License. See [LICENSE.txt](/LICENSE.txt).
