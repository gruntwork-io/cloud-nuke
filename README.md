# aws-nuke

This repo contains a CLI tool to delete all AWS resources in an account. The currently supported functionality includes:

* Deleting all unprotected EC2 instances in an AWS account

## Install

1. Download the latest binary for your OS on the [releases page](https://github.com/gruntwork-io/aws-nuke/releases).
2. Move the binary to a folder on your `PATH`. E.g.: `mv gruntwork_darwin_amd64 /usr/local/bin/gruntwork`.
3. Add execute permissions to the binary. E.g.: `chmod u+x /usr/local/bin/aws-nuke`.
4. Test it installed correctly: `aws-nuke --help`.

## Usage

Simply running `aws-nuke` will start the process of cleaning up your AWS account. You'll be shown a list of resources that'll be deleted as well as a prompt to confirm before any deletion actually takes place.

Happy Nuking!!!

## Credentials

In order for the `aws-nuke` CLI tool to access your AWS, you will need to provide your AWS credentials. You can used one of the [standard AWS CLI credential mechanisms](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html).

## Running Tests

```shell
go test -v ./...
```
