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

Cloud-nuke suppports 🔎 inspecting and 🔥💀 deleting the following AWS resources:

| Resource Family         | Resource type                                            |
|-------------------------|----------------------------------------------------------|
| App Runner              | Service                                                  |
| Data Sync               | Location                                                 |
| Data Sync               | Task                                                     |
| EC2                     | Auto scaling groups                                      |
| EC2                     | Elastic Load Balancers (v1 and v2)                       |
| EC2                     | EBS Volumes                                              |
| EC2                     | Unprotected EC2 instances                                |
| EC2                     | AMIS                                                     |
| EC2                     | Snapshots                                                |
| EC2                     | Elastic IPs                                              |
| EC2                     | Launch Configurations                                    |
| EC2                     | IPAM (Amazon VPC IP Address Manager)                     |
| EC2                     | IPAM Pool                                                |
| EC2                     | IPAM Scope                                               |
| EC2                     | IPAM Custom Allocation                                   |
| EC2                     | IPAM BYOASN                                              |
| EC2                     | IPAM Resource Discovery                                  |
| EC2                     | Internet Gateway                                         |
| EC2                     | Network ACL                                              |
| EC2                     | Egress only internet gateway                             |
| EC2                     | Endpoint                                                 |
| EC2                     | Security Group                                           |
| EC2                     | Network Interface                                        |
| EC2                     | Placement Group                                          |
| Certificate Manager     | ACM Private CA                                           |
| Direct Connect          | Transit Gateways                                         |
| Elasticache             | Clusters                                                 |
| Elasticache             | Parameter Groups                                         |
| Elasticache             | Subnet Groups                                            |
| Elastic Beanstalk       | Applications                                             |
| ECS                     | Services                                                 |
| ECS                     | Clusters                                                 |
| EKS                     | Clusters                                                 |
| DynamoDB                | Tables                                                   |
| Lambda                  | Functions                                                |
| SQS                     | Queues                                                   |
| S3                      | Buckets                                                  |
| S3                      | Access Points                                            |
| S3                      | Object Lambda Access Points                              |
| S3                      | Multi Region Access Points                               |
| VPC                     | Default VPCs                                             |
| VPC                     | Default rules in the un-deletable default security group |
| VPC                     | NAT Gateways                                             |
| IAM                     | Users                                                    |
| IAM                     | Roles (and any associated EC2 instance profiles)         |
| IAM                     | Service-linked-roles                                     |
| IAM                     | Groups                                                   |
| IAM                     | Policies                                                 |
| IAM                     | Customer-managed policies                                |
| IAM                     | Access analyzers                                         |
| IAM                     | OpenID Connect providers                                 |
| Secrets Manager         | Secrets                                                  |
| CloudWatch              | Dashboard                                                |
| CloudWatch              | Log groups                                               |
| CloudWatch              | Alarms                                                   |
| OpenSearch              | Domains                                                  |
| KMS                     | Custgomer managed keys (and associated key aliases)      |
| Managed Prometheus      | Prometheus Workspace                                     |
| GuardDuty               | Detectors                                                |
| Macie                   | Member accounts                                          |
| SageMaker               | Notebook instances                                       |
| Kinesis                 | Streams                                                  |
| Kinesis                 | Firehose                                                 |
| API Gateway             | Gateways (v1 and v2)                                     |
| EFS                     | File systems                                             |
| SNS                     | Topics                                                   |
| CloudTrail              | Trails                                                   |
| ECR                     | Repositories                                             |
| Config                  | Service recorders                                        |
| Config                  | Service rules                                            |
| RDS                     | RDS databases                                            |
| RDS                     | Neptune                                                  |
| RDS                     | Document DB instances                                    |
| RDS                     | RDS parameter group                                      |
| RDS                     | RDS Proxy                                                |
| Security Hub            | Hubs                                                     |
| Security Hub            | Members                                                  |
| Security Hub            | Administrators                                           |
| SES                     | SES configuration set                                    |
| SES                     | SES email template                                       |
| SES                     | SES Identity                                             |
| SES                     | SES receipt rule set                                     |
| SES                     | SES receipt filter                                       |
| AWS Certificate Manager | Certificates                                             |
| CodeDeploy              | Applications                                             |
| Route53                 | Hosted Zones                                             |
| Route53                 | CIDR collections                                         |
| Route53                 | Traffic Policies                                         |
| NetworkFirewall         | Network Firewall                                         |
| NetworkFirewall         | Network Firewall Policy                                  |
| NetworkFirewall         | Network Firewall Rule Group                              |
| NetworkFirewall         | Network Firewall TLS inspection configuration            |
| NetworkFirewall         | Network Firewall Resource Policy                         |
| VPCLattice              | VPC Lattice Service                                      |
| VPCLattice              | VPC Lattice Service Network                              |
| VPCLattice              | VPC Lattice Target Group                                 |

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
the values are the rules for which resources to terminate.

### Filtering Features

For each resource type, you can specify either `include` or `exclude` rules. Each rule can be one of the following
filters mentioned below. Here is an example:

```
s3:
  include:
    ...
  exclude:
    ...
```

#### Names Regex Filter

Now given the following config, the s3 buckets that will be nuked are further filtered to only include ones that match
any of the provided regular expressions. So a bucket named `alb-app-access-logs` would be deleted, but a bucket
named `my-s3-bucket` would not.

```yaml
s3:
  include:
    names_regex:
      - ^alb-.*-access-logs$
      - .*-prod-alb-.*
```

Similarly, you can adjust the config to delete only IAM users of a particular name by using the `IAMUsers` key. For
example, in the following config, only IAM users that have the prefix `my-test-user-` in their username will be deleted.

```yaml
IAMUsers:
  include:
    names_regex:
      - ^my-test-user-.*
```

Now consider the following contrived example:

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

The intention is to delete all the s3 buckets that match the include rules but not the exclude rules. Filtering is
**commutative**, meaning that you should get the same result whether you apply the include filters before or after the
exclude filters.

The result of these filters applied in either order will be a set of s3 buckets that match `^alb-.*-access-logs$` as
long as they do not also contain `public` or `prod`. The rule to include s3 buckets matching `.*-prod-alb-.*` is negated
by the rule to exclude those matching `prod`.

#### Time Filter

You can also filter resources by time. The following config will delete all s3 buckets that were created after
`2020-01-01T00:00:00Z`.

```yaml
s3:
  include:
    time_after: '2020-01-01T00:00:00Z'
```

Similarly, you can delete all s3 buckets that were created before `2020-01-01T00:00:00Z` by using the `time_before`

```yaml
s3:
  include:
    time_before: '2020-01-01T00:00:00Z'
```

#### Tag Filter

You can also exclude resources by tags. The following config will exclude all s3 buckets that have a tag with key `foo`
and value `true` (case-insensitive).

```yaml
s3:
  exclude:
    tag: 'foo'
```
#### Timeout
You have the flexibility to set individual timeout options for specific resources. The execution will pause until the designated timeout is reached for each resource.
```yaml
s3:
  timeout: 10m

  ........

s3:
  timeout: 5s

```

By default, it will use the exclusion default tag: `cloud-nuke-excluded` to exclude resources.
_Note: it doesn't support including resources by tags._

### What's supported?

To find out what we options are supported in the config file today, consult this table. Resource types at the top level
of the file that are supported are listed here.

| resource type                    | config key                    | names_regex                           | time                                | tags | timeout |
|----------------------------------|-------------------------------|---------------------------------------|-------------------------------------|------|---------|
| acm                              | ACM                           | ✅ (Domain Name)                       | ✅ (Created Time)                    | ❌    | ✅       |
| acmpca                           | ACMPCA                        | ❌                                     | ✅ (LastStateChange or Created Time) | ❌    | ✅       |
| ami                              | AMI                           | ✅ (Image Name)                        | ✅ (Creation Time)                   | ❌    | ✅       |
| apigateway                       | APIGateway                    | ✅ (API Name)                          | ✅ (Created Time)                    | ❌    | ✅       |
| apigatewayv2                     | APIGatewayV2                  | ✅ (API Name)                          | ✅ (Created Time)                    | ❌    | ✅       |
| accessanalyzer                   | AccessAnalyzer                | ✅ (Analyzer Name)                     | ✅ (Created Time)                    | ❌    | ✅       |
| asg                              | AutoScalingGroup              | ✅ (ASG Name)                          | ✅ (Created Time)                    | ✅    | ✅       |
| app-runner-service               | AppRunnerService              | ✅ (App Runner Service Name)           | ✅ (Created Time)                    | ❌    | ✅       |
| backup-vault                     | BackupVault                   | ✅ (Backup Vault Name)                 | ✅ (Created Time)                    | ❌    | ✅       |
| cloudwatch-alarm                 | CloudWatchAlarm               | ✅ (Alarm Name)                        | ✅ (AlarmConfigurationUpdated Time)  | ❌    | ✅       |
| cloudwatch-dashboard             | CloudWatchDashboard           | ✅ (Dashboard Name)                    | ✅ (LastModified Time)               | ❌    | ✅       |
| cloudwatch-loggroup              | CloudWatchLogGroup            | ✅ (Log Group Name)                    | ✅ (Creation Time)                   | ❌    | ✅       |
| cloudtrail                       | CloudtrailTrail               | ✅ (Trail Name)                        | ❌                                   | ❌    | ✅       |
| codedeploy-application           | CodeDeployApplications        | ✅ (Application Name)                  | ✅ (Creation Time)                   | ❌    | ✅       |
| config-recorders                 | ConfigServiceRecorder         | ✅ (Recorder Name)                     | ❌                                   | ❌    | ✅       |
| config-rules                     | ConfigServiceRule             | ✅ (Rule Name)                         | ❌                                   | ❌    | ✅       |
| data-sync-location               | DataSyncLocation              | ❌                                     | ❌                                   | ❌    | ✅       |
| data-sync-task                   | DataSyncTask                  | ✅ (Task Name)                         | ❌                                   | ❌    | ✅       |
| dynamodb                         | DynamoDB                      | ✅ (Table Name)                        | ✅ (Creation Time)                   | ❌    | ✅       |
| ebs                              | EBSVolume                     | ✅ (Volume Name)                       | ✅ (Creation Time)                   | ✅    | ✅       |
| elastic-beanstalk                | ElasticBeanstalk              | ✅ (Application Name)                  | ✅ (Creation Time)                   | ❌    | ✅       |
| ec2                              | EC2                           | ✅ (Instance Name)                     | ✅ (Launch Time)                     | ✅    | ✅       |
| ec2-dedicated-hosts              | EC2DedicatedHosts             | ✅ (EC2 Name Tag)                      | ✅ (Allocation Time)                 | ❌    | ✅       |
| ec2-dhcp-option                  | EC2DhcpOption                 | ❌                                     | ❌                                   | ❌    | ✅       |
| ec2-keypairs                     | EC2KeyPairs                   | ✅ (Key Pair Name)                     | ✅ (Creation Time)                   | ✅    | ✅       |
| ec2-ipam                         | EC2IPAM                       | ✅ (IPAM name)                         | ✅ (Creation Time)                   | ✅    | ✅       |
| ec2-ipam-pool                    | EC2IPAMPool                   | ✅ (IPAM Pool name)                    | ✅ (Creation Time)                   | ✅    | ✅       |
| ec2-ipam-resource-discovery      | EC2IPAMResourceDiscovery      | ✅ (IPAM Discovery Name)               | ✅ (Creation Time)                   | ✅    | ✅       |
| ec2-ipam-scope                   | EC2IPAMScope                  | ✅ (IPAM Scope Name)                   | ✅ (Creation Time)                   | ✅    | ✅       |
| ec2-placement-groups             | EC2PlacementGroups | ✅ (Placement Group Name)                         | ✅ (First Seen Tag Time)             | ✅    | ✅       |
| ec2-subnet                       | EC2Subnet                     | ✅ (Subnet Name)                       | ✅ (Creation Time)                   | ✅    | ❌       |
| ec2-endpoint                     | EC2Endpoint                   | ✅ (Endpoint Name)                     | ✅ (Creation Time)                   | ✅    | ✅       |
| ecr                              | ECRRepository                 | ✅ (Repository Name)                   | ✅ (Creation Time)                   | ❌    | ✅       |
| ecscluster                       | ECSCluster                    | ✅ (Cluster Name)                      | ❌                                   | ❌    | ✅       |
| ecsserv                          | ECSService                    | ✅ (Service Name)                      | ✅ (Creation Time)                   | ❌    | ✅       |
| ekscluster                       | EKSCluster                    | ✅ (Cluster Name)                      | ✅ (Creation Time)                   | ✅    | ✅       |
| elb                              | ELBv1                         | ✅ (Load Balancer Name)                | ✅ (Created Time)                    | ❌    | ✅       |
| elbv2                            | ELBv2                         | ✅ (Load Balancer Name)                | ✅ (Created Time)                    | ❌    | ✅       |
| efs                              | ElasticFileSystem             | ✅ (File System Name)                  | ✅ (Creation Time)                   | ❌    | ✅       |
| eip                              | ElasticIP                     | ✅ (Elastic IP Allocation Name)        | ✅ (First Seen Tag Time)             | ✅    | ✅       |
| elasticache                      | Elasticache                   | ✅ (Cluster ID & Replication Group ID) | ✅ (Creation Time)                   | ❌    | ✅       |
| elasticacheparametergroups       | ElasticacheParameterGroups    | ✅ (Parameter Group Name)              | ❌                                   | ❌    | ✅       |
| elasticachesubnetgroups          | ElasticacheSubnetGroups       | ✅ (Subnet Group Name)                 | ❌                                   | ❌    | ✅       |
| guardduty                        | GuardDuty                     | ❌                                     | ✅ (Created Time)                    | ❌    | ✅       |
| iam-group                        | IAMGroups                     | ✅ (Group Name)                        | ✅ (Creation Time)                   | ❌    | ✅       |
| iam-policy                       | IAMPolicies                   | ✅ (Policy Name)                       | ✅ (Creation Time)                   | ❌    | ✅       |
| iam-role                         | IAMRoles                      | ✅ (Role Name)                         | ✅ (Creation Time)                   | ❌    | ✅       |
| iam-service-linked-role          | IAMServiceLinkedRoles         | ✅ (Service Linked Role Name)          | ✅ (Creation Time)                   | ❌    | ✅       |
| iam                              | IAMUsers                      | ✅ (User Name)                         | ✅ (Creation Time)                   | ✅    | ✅       |
| internet-gateway                 | InternetGateway               | ✅ (Gateway Name)                      | ✅ (Creation Time)                   | ✅    | ✅       |
| egress-only-internet-gateway     | EgressOnlyInternetGateway     | ✅ (Gateway name)                      | ✅ (Creation Time)                   | ✅    | ✅       |
| kmscustomerkeys                  | KMSCustomerKeys               | ✅ (Key Name)                          | ✅ (Creation Time)                   | ❌    | ❌       |
| kinesis-stream                   | KinesisStream                 | ✅ (Stream Name)                       | ❌                                   | ❌    | ✅       |
| kinesis-firehose                 | KinesisFirehose               | ✅ (Delivery Stream Name)              | ❌                                   | ❌    | ✅       |
| lambda                           | LambdaFunction                | ✅ (Function Name)                     | ✅ (Last Modified Time)              | ❌    | ✅       |
| lc                               | LaunchConfiguration           | ✅ (Launch Configuration Name)         | ✅ (Created Time)                    | ❌    | ✅       |
| lt                               | LaunchTemplate                | ✅ (Launch Template Name)              | ✅ (Created Time)                    | ❌    | ✅       |
| macie-member                     | MacieMember                   | ❌                                     | ✅ (Creation Time)                   | ❌    | ✅       |
| msk-cluster                      | MSKCluster                    | ✅ (Cluster Name)                      | ✅ (Creation Time)                   | ❌    | ✅       |
| managed-prometheus               | ManagedPrometheus             | ✅ (Workspace Alias)                   | ✅ (Creation Time)                   | ✅    | ✅       |
| nat-gateway                      | NatGateway                    | ✅ (EC2 Name Tag)                      | ✅ (Creation Time)                   | ✅    | ✅       |
| network-acl                      | NetworkACL                    | ✅ (ACL Name Tag)                      | ✅ (Creation Time)                   | ✅    | ✅       |
| network-interface                | NetworkInterface              | ✅ (Interface Name Tag)                | ✅ (Creation Time)                   | ✅    | ✅       |
| oidcprovider                     | OIDCProvider                  | ✅ (Provider URL)                      | ✅ (Creation Time)                   | ❌    | ✅       |
| opensearchdomain                 | OpenSearchDomain              | ✅ (Domain Name)                       | ✅ (First Seen Tag Time)             | ❌    | ✅       |
| redshift                         | Redshift                      | ✅ (Cluster Identifier)                | ✅ (Creation Time)                   | ❌    | ✅       |
| rds-cluster                      | DBClusters                    | ✅ (DB Cluster Identifier )            | ✅ (Creation Time)                   | ✅    | ✅       |
| rds                              | DBInstances                   | ✅ (DB Name)                           | ✅ (Creation Time)                   | ✅    | ✅       |
| rds-parameter-group              | RdsParameterGroup             | ✅ (Group Name)                        | ❌                                   | ❌    | ✅       |
| rds-subnet-group                 | DBSubnetGroups                | ✅ (DB Subnet Group Name)              | ❌                                   | ❌    | ✅       |
| rds-proxy                        | RDSProxy                      | ✅ (proxy Name)                        | ✅ (Creation Time)                   | ❌    | ✅       |
| s3                               | s3                            | ✅ (Bucket Name)                       | ✅ (Creation Time)                   | ✅    | ✅       |
| s3-ap                            | s3AccessPoint                 | ✅ (Access point Name)                 | ❌                                   | ❌    | ✅       |
| s3-olap                          | S3ObjectLambdaAccessPoint     | ✅ (Object Lambda Access point Name)   | ❌                                   | ❌    | ✅       |
| s3-mrap                          | S3MultiRegionAccessPoint      | ✅ (Multi region Access point Name)    | ✅ (Creation Time)                   | ❌    | ✅       |
| security-group                   | SecurityGroup                 | ✅ (Security group name)               | ✅ (Creation Time)                   | ✅    | ❌       |
| ses-configuration-set            | SesConfigurationset           | ✅ (Configuration set name)            | ❌                                   | ❌    | ✅       |
| ses-email-template               | SesEmailTemplates             | ✅ (Template Name)                     | ✅ (Creation Time)                   | ❌    | ✅       |
| ses-identity                     | SesIdentity                   | ✅ (Identity -Mail/Domain)             | ❌                                   | ❌    | ✅       |
| ses-receipt-rule-set             | SesReceiptRuleSet             | ✅ (Receipt Rule Set Name)             | ✅ (Creation Time)                   | ❌    | ✅       |
| ses-receipt-filter               | SesReceiptFilter              | ✅ (Receipt Filter Name)               | ❌                                   | ❌    | ✅       |
| snstopic                         | SNS                           | ✅ (Topic Name)                        | ✅ (First Seen Tag Time)             | ❌    | ✅       |
| sqs                              | SQS                           | ✅ (Queue Name)                        | ✅ (Creation Time)                   | ❌    | ✅       |
| sagemaker-notebook-smni          | SageMakerNotebook             | ✅ (Notebook Instnace Name)            | ✅ (Creation Time)                   | ❌    | ✅       |
| secretsmanager                   | SecretsManager                | ✅ (Secret Name)                       | ✅ (Last Accessed or Creation Time)  | ❌    | ✅       |
| security-hub                     | SecurityHub                   | ❌                                     | ✅ (Created Time)                    | ❌    | ✅       |
| snap                             | Snapshots                     | ❌                                     | ✅ (Creation Time)                   | ✅    | ✅       |
| transit-gateway                  | TransitGateway                | ❌                                     | ✅ (Creation Time)                   | ❌    | ✅       |
| transit-gateway-route-table      | TransitGatewayRouteTable      | ❌                                     | ✅ (Creation Time)                   | ❌    | ✅       |
| transit-gateway-attachment       | TransitGatewaysVpcAttachment  | ❌                                     | ✅ (Creation Time)                   | ❌    | ✅       |
| vpc                              | VPC                           | ✅ (EC2 Name Tag)                      | ✅ (First Seen Tag Time)             | ❌    | ❌       |
| route53-hosted-zone              | Route53HostedZone             | ✅ (Hosted zone name)                  | ❌                                   | ❌    | ❌       |
| route53-cidr-collection          | Route53CIDRCollection         | ✅ (Cidr collection name)              | ❌                                   | ❌    | ❌       |
| route53-traffic-policy           | Route53TrafficPolicy          | ✅ (Traffic policy name)               | ❌                                   | ❌    | ❌       |
| network-firewall                 | NetworkFirewall               | ✅ (Firewall name)                     | ✅ (First Seen Tag Time)             | ✅    | ❌       |
| network-firewall-policy          | NetworkFirewallPolicy         | ✅ (Firewall Policy name)              | ✅ (First Seen Tag Time)             | ✅    | ❌       |
| network-firewall-rule-group      | NetworkFirewallRuleGroup      | ✅ (Firewall Rule group name)          | ✅ (First Seen Tag Time)             | ✅    | ❌       |
| network-firewall-tls-config      | NetworkFirewallTLSConfig      | ✅ (Firewall TLS config name)          | ✅ (First Seen Tag Time)             | ✅    | ❌       |
| network-firewall-resource-policy | NetworkFirewallResourcePolicy | ✅ (Firewall Resource Policy ARN)      | ❌                                   | ❌    | ❌       |
| vpc-lattice-service              | VPCLatticeService             | ✅ (VPC Lattice service ARN)           | ✅ (Creation Time)                   | ❌    | ✅       |
| vpc-lattice-service-network      | VPCLatticeServiceNetwork      | ✅ (VPC Lattice service network ARN)   | ✅ (Creation Time)                   | ❌    | ✅       |
| vpc-lattice-target-group         | VPCLatticeTargetGroup         | ✅ (VPC Lattice target group ARN)      | ✅ (Creation Time)                   | ❌    | ✅       |

### Resource Deletion and 'IsNukable' Check Option
#### Supported Resources for 'IsNukable' Check
For certain resources, such as `AMI`, `EBS`, `DHCP Option`, and others listed below, we support an option to verify whether the user has sufficient permissions to nuke the resources. If not, it will raise `error: INSUFFICIENT_PERMISSION` error.

Supported resources:
- AMI
- EBS
- DHCP Option
- Egress only Internet Gateway
- Endpoints
- Internet Gatway
- IPAM
- IPAM BYOASN
- IPAM Custom Allocation
- IPAM Pool
- IPAM Resource Discovery
- IPAM Scope
- Key Pair
- Network ACL
- Network Interface
- Subnet
- VPC
- Elastic IP
- Launch Template
- NAT Gateway
- Network Firewall
- Security Group
- SnapShot
- Transit Gateway

#### Unsupported Resources
Please note that the eligibility check for nukability relies on the `DryRun` feature provided by AWS. Regrettably, this feature is not available for all delete APIs of resource types. Hence, the 'eligibility check for nukability' option may not be accessible for all resource types


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
one of
the [standard AWS CLI credential mechanisms](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html).

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
