# cloud-nuke

This repo contains a CLI tool to delete all cloud (AWS, Azure, GCP) resources in an account. cloud-nuke was created for situations when you might have an account you use for testing and need to clean up left over resources so you're not charged for them. Also great for cleaning out accounts with redundant resources.

The currently supported functionality includes:

## AWS

* Deleting all Auto scaling groups in an AWS account
* Deleting all Elastic Load Balancers (Classic and V2) in an AWS account
* Deleting all EBS Volumes in an AWS account
* Deleting all unprotected EC2 instances in an AWS account
* Deleting all AMIs in an AWS account
* Deleting all Snapshots in an AWS account
* Deleting all Elastic IPs in an AWS account

## Azure

_Coming Soon_

## GCP

_Coming Soon_

### WARNING: THIS TOOL IS HIGHLY DESTRUCTIVE, ALL SUPPORTED RESOURCES WILL BE DELETED. ITS EFFECTS ARE IRREVERSIBLE AND SHOULD NEVER BE USED IN A PRODUCTION ENVIRONMENT

## Install

1. Download the latest binary for your OS on the [releases page](https://github.com/gruntwork-io/cloud-nuke/releases).
2. Move the binary to a folder on your `PATH`. E.g.: `mv cloud-nuke_darwin_amd64 /usr/local/bin/cloud-nuke`.
3. Add execute permissions to the binary. E.g.: `chmod u+x /usr/local/bin/cloud-nuke`.
4. Test it installed correctly: `cloud-nuke --help`.

## Usage

Simply running `cloud-nuke <provider>` (e.g. `cloud-nuke aws`) will start the process of cleaning up your cloud account. You'll be shown a list of resources that'll be deleted as well as a prompt to confirm before any deletion actually takes place.

### Excluding Regions

You can use the `--exclude-region` flag to exclude resources in certain regions from being deleted. For example the following command does not nuke resources in `ap-south-1` and `ap-south-2` regions:

```shell
cloud-nuke aws --exclude-region ap-south-1 --exclude-region ap-south-2
```

### Excluding Resources by Age

You can use the `--older-than` flag to only nuke resources that were created before a certain period, the possible values are all valid values for [ParseDuration](https://golang.org/pkg/time/#ParseDuration) For example the following command nukes resources that are at least one day old:

```shell
cloud-nuke aws --older-than 24h
```

Happy Nuking!!!

## Credentials

### AWS

In order for the `cloud-nuke` CLI tool to access your AWS, you will need to provide your AWS credentials. You can used one of the [standard AWS CLI credential mechanisms](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html).

## Running Tests

```shell
go test -v ./...
```

## License

This code is released under the MIT License. See [LICENSE.txt](/LICENSE.txt).
