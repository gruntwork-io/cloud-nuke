[![Maintained by Gruntwork.io](https://img.shields.io/badge/maintained%20by-gruntwork.io-%235849a6.svg)](https://gruntwork.io/?ref=repo_cloud_nuke)

# cloud-nuke

This repo contains a CLI tool to delete all resources in an AWS account. cloud-nuke was created for situations when you might have an account you use for testing and need to clean up leftover resources so you're not charged for them. Also great for cleaning out accounts with redundant resources. Also great for removing unnecessary defaults like default VPCs and permissive ingress/egress rules in default security groups.

In addition, cloud-nuke offers non-destructive inspecting functionality that can either be called via the command-line interface, or consumed as library methods, for scripting purposes.

The currently supported functionality includes:

## AWS

- Inspecting and deleting all ACM Private CA in an AWS account
- Inspecting and deleting all Auto scaling groups in an AWS account
- Inspecting and deleting all Elastic Load Balancers (v1 and v2) in an AWS account
- Inspecting and deleting all Transit Gateways in an AWS account
- Inspecting and deleting all EBS Volumes in an AWS account
- Inspecting and deleting all unprotected EC2 instances in an AWS account
- Inspecting and deleting all AMIs in an AWS account
- Inspecting and deleting all Snapshots in an AWS account
- Inspecting and deleting all Elastic IPs in an AWS account
- Inspecting and deleting all Elasticache clusters in an AWS account
- Inspecting and deleting all Launch Configurations in an AWS account
- Inspecting and deleting all ECS services in an AWS account
- Inspecting and deleting all ECS clusters in an AWS account
- Inspecting and deleting all EKS clusters in an AWS account
- Inspecting and deleting all RDS DB instances in an AWS account
- Inspecting and deleting all Lambda Functions in an AWS account
- Inspecting and deleting all SQS queues in an AWS account
- Inspecting and deleting all S3 buckets in an AWS account - except for buckets tagged with Key=cloud-nuke-excluded Value=true
- Inspecting and deleting all default VPCs in an AWS account
- Deleting VPCs in an AWS Account (along with any dependency resources such as ENIs, Egress Only Gateways, and Security Groups. except for default VPCs which is handled by the dedicated `defaults-aws` subcommand)
- Inspecting and deleting all IAM users in an AWS account
- Inspecting and deleting all IAM roles (and any associated EC2 instance profiles) in an AWS account
- Inspecting and deleting all IAM groups in an AWS account
- Inspecting and deleting all customer managed IAM policies in an AWS account
- Inspecting and deleting all Secrets Manager Secrets in an AWS account
- Inspecting and deleting all NAT Gateways in an AWS account
- Inspecting and deleting all IAM Access Analyzers in an AWS account
- Revoking the default rules in the un-deletable default security group of a VPC
- Inspecting and deleting all DynamoDB tables in an AWS account
- Inspecting and deleting all CloudWatch Dashboards in an AWS account
- Inspecting and deleting all OpenSearch Domains in an AWS account
- Inspecting and deleting all IAM OpenID Connect Providers
- Inspecting and deleting all Customer managed keys (and associated key aliases) from Key Management Service in an AWS account
- Inspecting and deleting all CloudWatch Log Groups in an AWS Account
- Inspecting and deleting all GuardDuty Detectors in an AWS Account
- Inspecting and deleting all Macie member accounts in an AWS account - as long as those accounts were created by Invitation - and not via AWS Organizations
- Inspecting and deleting all SageMaker Notebook Instances in an AWS account
- Inspecting and deleting all Kinesis Streams in an AWS account
- Inspecting and deleting all API Gateways (v1 and v2) in an AWS account
- Inspecting and deleting all Elastic FileSystems (efs) in an AWS account
- Inspecting and deleting all SNS Topics in an AWS account
- Inspecting and deleting all CloudTrail Trails in an AWS account

### BEWARE!

When executed as `cloud-nuke aws`, this tool is **HIGHLY DESTRUCTIVE** and deletes all resources! This mode should never be used in a production environment!

When executed as `cloud-nuke defaults-aws`, this tool deletes all DEFAULT VPCs and the default ingress/egress rule for all default security groups. This should be used in production environments **WITH CAUTION**.

## Install

### Download from releases page

1. Download the latest binary for your OS on the [releases page](https://github.com/gruntwork-io/cloud-nuke/releases).
2. Move the binary to a folder on your `PATH`. E.g.: `mv cloud-nuke_darwin_amd64 /usr/local/bin/cloud-nuke`.
3. Add execute permissions to the binary. E.g.: `chmod u+x /usr/local/bin/cloud-nuke`.
4. Test it installed correctly: `cloud-nuke --help`.

### Install via package manager

Note that package managers are third party. The third party cloud-nuke packages may not be updated with the latest version, but are often close. Please check your version against the latest available on the [releases page](https://github.com/gruntwork-io/cloud-nuke/releases). If you want the latest version, the recommended installation option is to [download from the releases page](https://github.com/gruntwork-io/cloud-nuke/releases).

- **macOS:** You can install cloud-nuke using [Homebrew](https://brew.sh/): `brew install cloud-nuke`.

- **Linux:** Most Linux users can use [Homebrew](https://docs.brew.sh/Homebrew-on-Linux): `brew install cloud-nuke`.

## Usage

Simply running `cloud-nuke aws` will start the process of cleaning up your cloud account. You'll be shown a list of resources that'll be deleted as well as a prompt to confirm before any deletion actually takes place.

In AWS, to delete only the default resources, run `cloud-nuke defaults-aws`. This will remove the default VPCs in each region, and will also revoke the ingress and egress rules associated with the default security group in each VPC. Note that the default security group itself is unable to be deleted.

### Nuke or inspect resources in certain regions

When using `cloud-nuke aws`, or `cloud-nuke inspect-aws`, you can use the `--region` flag to target resources in certain regions. For example the following command will nuke resources only in `ap-south-1` and `ap-south-2` regions:

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

When using `cloud-nuke aws` or `cloud-nuke inspect-aws`, you can use the `--exclude-region` flag to exclude resources in certain regions from being deleted or inspected. For example the following command does not nuke resources in `ap-south-1` and `ap-south-2` regions:

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

You can use the `--older-than` flag to only nuke resources that were created before a certain period, the possible values are all valid values for [ParseDuration](https://golang.org/pkg/time/#ParseDuration) For example the following command nukes resources that are at least one day old:

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

Just like you can select which resources to terminate using `--resource-type`, you can select which resources to skip using
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



### Using cloud-nuke as a library

You can import cloud-nuke into other projects and use it as a library for programmatically inspecting and counting resources.

```golang

package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	nuke_aws "github.com/gruntwork-io/cloud-nuke/aws"
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

	// Any custom settings you want
	myCustomConfig := &aws.Config{}

	myCustomConfig.WithMaxRetries(3)
	myCustomConfig.WithLogLevel(aws.LogDebugWithRequestErrors)
	// Optionally, set custom credentials
	// myCustomConfig.WithCredentials()

	// Be sure to set your config prior to calling any library methods such as NewQuery
	externalcreds.Set(myCustomConfig)

	// NewQuery is a convenience method for configuring parameters you want to pass to your resource search
	query, err := nuke_aws.NewQuery(
		targetRegions,
		excludeRegions,
		resourceTypes,
		excludeResourceTypes,
		excludeAfter,
	)
	if err != nil {
		fmt.Println(err)
	}

	// InspectResources still returns *AwsAccountResources, but this struct has been extended with several
	// convenience methods for quickly determining if resources exist in a given region
	accountResources, err := nuke_aws.InspectResources(query)
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


### Config file

For more granularity, such as being able to specify which resources to terminate using regular expressions or plain text, you can pass in a configuration file.

_Note: Config file support is a new feature and only filtering a handful of resources by name using regular expressions is currently supported. We'll be adding more support in the future, and pull requests are welcome!_

The following resources support the Config file:

- S3 Buckets
    - Resource type: `s3`
    - Config key: `s3`
- IAM Users
    - Resource type: `iam`
    - Config key: `IAMUsers`
- Secrets Manager Secrets
    - Resource type: `secretsmanager`
    - Config key: `SecretsManager`
- NAT Gateways
    - Resource type: `nat-gateway`
    - Config key: `NatGateway`
- IAM Access Analyzers
    - Resource type: `accessanalyzer`
    - Config key: `AccessAnalyzer`
- CloudWatch Dashboards
    - Resource type: `cloudwatch-dashboard`
    - Config key: `CloudWatchDashboard`
- OpenSearch Domains
    - Resource type: `opensearch`
    - Config key: `OpenSearchDomain`
- DynamoDB Tables
    - Resource type: `dynamodb`
    - Config key: `DynamoDB`
- EBS Volumes
    - Resource type: `ebs`
    - Config key: `EBSVolume`
- Lambda Functions
    - Resource type: `lambda`
    - Config key: `LambdaFunction`
- Elastic Load Balancers
    - Resource type: `elbv2`
    - Config key: `ELBv2`
- ECS Services
    - Resource type: `ecsserv`
    - Config key: `ECSService`
- ECS Clusters
    - Resource type: `ecscluster`
    - Config key: `ECSCluster`
- Elasticache
    - Resource type: `elasticache`
    - Config key: `Elasticache`
- VPCs
    - Resource type: `vpc`
    - Config key: `VPC`
- IAM OpenID Connect Providers
    - Resource type: `oidcprovider`
    - Config key: `OIDCProvider`
- CloudWatch LogGroups
    - Resource type: `cloudwatch-loggroup`
    - Config key: `CloudWatchLogGroup`
- KMS customer keys
    - Resource type: `kmscustomerkeys`
    - Config key: `KMSCustomerKeys`
- Auto Scaling Groups
    - Resource type: `asg`
    - Config key: `AutoScalingGroup`
- Launch Configurations
    - Resource type: `lc`
    - Config key: `LaunchConfiguration`
- Elastic IP Address
    - Resource type: `eip`
    - Config key: `ElasticIP`
- EC2 Instances
    - Resource type: `ec2`
    - Config key: `EC2`
- EC2 Key Pairs
  - Resource type: `ec2-keypairs`
  - Config key: `EC2KeyPairs`
- EKS Clusters
    - Resource type: `ekscluster`
    - Config key: `EKSCluster`
- SageMaker Notebook Instances
  - Resource type: `sagemaker-notebook-instances`
  - Config key: `SageMakerNotebook`
- API Gateways (v1) 
  - Resource type: `apigateway`
  - Config key: `APIGateway`
- API Gateways (v2) 
  - Resource type: `apigatewayv2`
  - Config key: `APIGatewayV2`
- Elastic FileSystems (efs) 
  - Resource type: `efs`
  - Config key: `ElasticFileSystem`






Notes:
  * no configuration options for KMS customer keys, since keys are created with auto-generated identifier

- Kinesis Streams
    - Resource type: `kinesis-stream`
    - Config key: `KinesisStream`

#### Example

```shell
cloud-nuke aws --resource-type s3 --config path/to/file.yaml
```

Given this command, `cloud-nuke` will nuke _only_ S3 buckets, as specified by the `--resource-type s3` option.

Now given the following config, the s3 buckets that will be nuked are further filtered to only include ones that match any of the provided regular expressions. So a bucket named `alb-app-access-logs` would be deleted, but a bucket named `my-s3-bucket` would not.
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

#### Include and exclude together

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

The intention is to delete all the s3 buckets that match the include rules but not the exclude rules. Filtering is commutative, meaning that you should get the same result whether you apply the include filters before or after the exclude filters.

The result of these filters applied in either order will be a set of s3 buckets that match `^alb-.*-access-logs$` as long as they do not also contain `public` or `prod`. The rule to include s3 buckets matching `.*-prod-alb-.*` is negated by the rule to exclude those matching `prod`.

<!-- We might only want to support region and resource-type in the command line, rather than in the config file.

Given this config, `cloud-nuke` will nuke all S3 buckets that exist in `us-east-1` and all S3 buckets that exist in `us-west-1`.
```yaml
s3:
  include:
    regions:
      - us-east-1
      - us-west-1
```

Given this config, `cloud-nuke` will nuke all S3 buckets that match the regular expression but only if they do not also exist in `us-east-1`. So a bucket named `abc-prod-alb-def` located in the `ap-northeast-2` region would be nuked.
```yaml
s3:
  include:
    names_regex:
      - .*-prod-alb-.*
  exclude:
    regions:
      - us-east-1
```
-->


#### CLI options override config file options

The options provided in the command line take precedence over those provided in any config file that gets passed in. For example, say you provide `--resource-type s3` in the command line, along with a config file that specifies `ec2:` at the top level but doesn't specify `s3:`. The command line argument filters the resource types to include only s3, so the rules in the config file for `ec2:` are ignored, and ec2 resources are not nuked. All s3 resources would be nuked.

In the same vein, say you do not provide a `--resource-type` option in the command line, but you do pass in a config file that only lists rules for `s3:`, such as `cloud-nuke aws --config path/to/config.yaml`. In this case _all_ resources would be nuked, but among `s3` buckets, only those matching your config file rules would be nuked.

Be careful when nuking and append the `--dry-run` option if you're unsure. Even without `--dry-run`, `cloud-nuke` will list resources that would undergo nuking and wait for your confirmation before carrying it out.

#### What's supported?

To find out what we options are supported in the config file today, consult this table. Resource types at the top level of the file that are supported are listed here.

| resource type                | names | names_regex | tags | tags_regex |
|------------------------------|-------|-------------|------|------------|
| s3                           | none  | ✅           | none | none       |
| iam user                     | none  | ✅           | none | none       |
| ecsserv                      | none  | ✅           | none | none       |
| ecscluster                   | none  | ✅           | none | none       |
| secretsmanager               | none  | ✅           | none | none       |
| nat-gateway                  | none  | ✅           | none | none       |
| accessanalyzer               | none  | ✅           | none | none       |
| dynamodb                     | none  | ✅           | none | none       |
| ebs                          | none  | ✅           | none | none       |
| lambda                       | none  | ✅           | none | none       |
| elbv2                        | none  | ✅           | none | none       |
| ecs                          | none  | ✅           | none | none       |
| elasticache                  | none  | ✅           | none | none       |
| vpc                          | none  | ✅           | none | none       |
| oidcprovider                 | none  | ✅           | none | none       |
| cloudwatch-loggroup          | none  | ✅           | none | none       |
| kmscustomerkeys              | none  | ✅           | none | none       |
| asg                          | none  | ✅           | none | none       |
| lc                           | none  | ✅           | none | none       |
| eip                          | none  | ✅           | none | none       |
| ec2                          | none  | ✅           | none | none       |
| apigateway                   | none  | ✅           | none | none       |
| apigatewayv2                 | none  | ✅           | none | none       |
| eks                          | none  | ✅           | none | none       |
| kinesis-stream               | none  | ✅           | none | none       |
| efs                          | none  | ✅           | none | none       |
| acmpca                       | none  | none        | none | none       |
| iam role                     | none  | ✅           | none | none       |
| iam policy                   | none  | ✅           | none | none       |
| sagemaker-notebook-instances | none  | ✅           | none | none       |
| ... (more to come)           | none  | none        | none | none       |


### Log level
By default, cloud-nuke sends most output to the `Debug` level logger, to enhance legibility, since the results of every deletion attempt will be displayed in the report that cloud-nuke prints after each run. 

However, sometimes it's helpful to see all output, such as when you're debugging something. 

You can set the log level by specifying the `--log-level` flag as per [logrus](https://github.com/sirupsen/logrus) log levels:

```shell
cloud-nuke aws --log-level debug
```

OR

```shell
LOG_LEVEL=debug cloud-nuke aws
```

Default value is - `info`. Acceptable values are `debug, info, warn, error, panic, fatal, trace` as per [logrus log level parser](https://github.com/sirupsen/logrus/blob/master/logrus.go#L25).

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
- Elastic IPs (Supported as a separate resource that gets cleaned up first. If you are filtering what gets nuked, Elastic IPs may prevent VPCs from destroying.)

All other resources that get created within VPCs must be cleaned up prior to running cloud-nuke on VPC resources.

Happy Nuking!!!

## Credentials

### AWS

In order for the `cloud-nuke` CLI tool to access your AWS, you will need to provide your AWS credentials. You can use one of the [standard AWS CLI credential mechanisms](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html).

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

**Note**: Many of the tests in the `aws` folder run against a real AWS account and will create and destroy actual resources. DO NOT
hit `CTRL+C` while the tests are running, as this will prevent them from cleaning up properly. We are not responsible for any
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

Use env-vars to opt-in to special tests, which are expensive to run:

```bash
# Run acmpca tests
TEST_ACMPCA_EXPENSIVE_ENABLE=1 go test -v ./...
```

### Formatting

Every source file in this project should be formatted with `go fmt`.

### Releasing new versions
We try to follow the release process as deifned in our [Coding Methodology](https://www.notion.so/gruntwork/Gruntwork-Coding-Methodology-02fdcd6e4b004e818553684760bf691e#08b68ee0e19143e89523dcf483d2bf48).

#### Choosing a new release tag
If the new release contains any new resources that `cloud-nuke` will support, mark it as a minor version bump (X in v0.X.Y)
to indicate backward incompatibilities.

This is because since version `v0.2.0` `cloud-nuke` has been configured to automatically include new resources (so you have
to explicitly opt-out). This is inherently not backward compatible, because users with CI practices around `cloud-nuke` would
be surprised by new resources that are suddenly being picked up for deletion! This surprise is more alarming for resources
that are actively in use for any account, such as IAM Users.

Therefore please mark your release as backward incompatible and bump the **minor version** (`X` in `v0.X.Y`) when it includes
support for nuking new resources, so that we provide better signals for users when we introduce a new resource.

#### To release a new version
Go to the [Releases Page](https://github.com/gruntwork-io/cloud-nuke/releases) and create a new release. The CircleCI job for this repo has been configured to:

1. Automatically detect new tags.
1. Build binaries for every OS using that tag as a version number.
1. Upload the binaries to the release in GitHub.

See `.circleci/config.yml` for details.

## License

This code is released under the MIT License. See [LICENSE.txt](/LICENSE.txt).
