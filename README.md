[![Maintained by Gruntwork.io](https://img.shields.io/badge/maintained%20by-gruntwork.io-%235849a6.svg)](https://gruntwork.io/?ref=repo_cloud_nuke)

# cloud-nuke

This repo contains a CLI tool to delete all resources in an AWS account. cloud-nuke was created for situations when you might have an account you use for testing and need to clean up leftover resources so you're not charged for them. Also great for cleaning out accounts with redundant resources. Also great for removing unnecessary defaults like default VPCs and permissive ingress/egress rules in default security groups.

The currently supported functionality includes:

## AWS

- Deleting all Auto scaling groups in an AWS account
- Deleting all Elastic Load Balancers (Classic and V2) in an AWS account
- Deleting all EBS Volumes in an AWS account
- Deleting all unprotected EC2 instances in an AWS account
- Deleting all AMIs in an AWS account
- Deleting all Snapshots in an AWS account
- Deleting all Elastic IPs in an AWS account
- Deleting all Launch Configurations in an AWS account
- Deleting all ECS services in an AWS account
- Deleting all EKS clusters in an AWS account
- Deleting all RDS DB instances in an AWS account
- Deleting all S3 buckets in an AWS account - except for buckets tagged with Key=cloud-nuke-excluded Value=true
- Deleting all default VPCs in an AWS account
- Revoking the default rules in the un-deletable default security group of a VPC

### Caveats

- We currently do not support deleting ECS clusters because AWS
  does not give us a good way to blacklist clusters off the list (there are no
  tags and we do not know the creation timestamp). Given the destructive nature
  of the tool, we have opted not to support deleting ECS clusters at the
  moment. See https://github.com/gruntwork-io/cloud-nuke/pull/36 for a more
  detailed discussion.

### BEWARE!

When executed as `cloud-nuke aws`, this tool is **HIGHLY DESTRUCTIVE** and deletes all resources! This mode should never be used in a production environment!

When executed as `cloud-nuke defaults-aws`, this tool deletes all DEFAULT VPCs and the default ingress/egress rule for all default security groups. This should be used in production environments **WITH CAUTION**.

## Install

1. Download the latest binary for your OS on the [releases page](https://github.com/gruntwork-io/cloud-nuke/releases).
2. Move the binary to a folder on your `PATH`. E.g.: `mv cloud-nuke_darwin_amd64 /usr/local/bin/cloud-nuke`.
3. Add execute permissions to the binary. E.g.: `chmod u+x /usr/local/bin/cloud-nuke`.
4. Test it installed correctly: `cloud-nuke --help`.

## Usage

Simply running `cloud-nuke aws` will start the process of cleaning up your cloud account. You'll be shown a list of resources that'll be deleted as well as a prompt to confirm before any deletion actually takes place.

In AWS, to delete only the default resources, run `cloud-nuke defaults-aws`. This will remove the default VPCs in each region, and will also revoke the ingress and egress rules associated with the default security group in each VPC. Note that the default security group itself is unable to be deleted.

### Nuke resources in certain regions

When using `cloud-nuke aws`, you can use the `--region` flag to target resources in certain regions for deletion. For example the following command will nuke resources only in `ap-south-1` and `ap-south-2` regions:

```shell
cloud-nuke aws --region ap-south-1 --region ap-south-2
```

Including regions is available within both `cloud-nuke aws` and with `cloud-nuke defaults-aws`.

### Exclude resources in certain regions

When using `cloud-nuke aws`, you can use the `--exclude-region` flag to exclude resources in certain regions from being deleted. For example the following command does not nuke resources in `ap-south-1` and `ap-south-2` regions:

```shell
cloud-nuke aws --exclude-region ap-south-1 --exclude-region ap-south-2
```

`--region` and `--exclude-region` flags cannot be specified together i.e. they are mutually exclusive.

Excluding regions is available within both `cloud-nuke aws` and with `cloud-nuke defaults-aws`.

### Excluding Resources by Age

You can use the `--older-than` flag to only nuke resources that were created before a certain period, the possible values are all valid values for [ParseDuration](https://golang.org/pkg/time/#ParseDuration) For example the following command nukes resources that are at least one day old:

```shell
cloud-nuke aws --older-than 24h
```

### List supported resource types

You can use the `--list-resource-types` flag to list resource types whose termination is currently supported:

```shell
cloud-nuke aws --list-resource-types
```

### Terminate specific resource types

If you want to target specific resource types (e.g ec2, ami, etc.) instead of all the supported resources you can
do so by specifying them through the `--resource-type` flag:

```shell
cloud-nuke aws --resource-type ec2 --resource-type ami
```

will search and target only `ec2` and `ami` resources. The specified resource type should be a valid resource type
i.e. it should be present in the `--list-resource-types` output. Using `--resource-type` also speeds up search because
we are searching only for specific resource types.

### Exclude terminating specific resource types

Just like you can select which resources to terminate using `--resource-type`, you can select which resources to skip using
`--exclude-resource-type` flag:

```shell
cloud-nuke aws --exclude-resource-type s3 --exclude-resource-type ec2
```

This will terminate all resource types other than S3 and EC2.

`--resource-type` and `--exclude-resource-type` flags cannot be specified together i.e. they are mutually exclusive.

### Dry run mode

If you want to check what resources are going to be targeted without actually terminating them, you can use the
`--dry-run` flag

```shell
cloud-nuke aws --resource-type ec2 --dry-run
```

### Nuking only default security group rules
When deleting defaults with `cloud-nuke defaults-aws`, use the `--sg-only` flag to delete only the default
security group rules and not the default VPCs.

```shell
cloud-nuke defaults-aws --sg-only
```

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

### Dependencies

- cloud-nuke uses `dep`, a vendor package management tool for golang. See the dep repo for
  [installation instructions](https://github.com/golang/dep). cloud-nuke currently does not support Go modules.

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

### Formatting

Every source file in this project should be formatted with `go fmt`.

### Releasing new versions

To release a new version, just go to the [Releases Page](https://github.com/gruntwork-io/cloud-nuke/releases) and
create a new release. The CircleCI job for this repo has been configured to:

1. Automatically detect new tags.
1. Build binaries for every OS using that tag as a version number.
1. Upload the binaries to the release in GitHub.

See `.circleci/config.yml` for details.

## License

This code is released under the MIT License. See [LICENSE.txt](/LICENSE.txt).
