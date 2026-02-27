[![Maintained by Gruntwork.io](https://img.shields.io/badge/maintained%20by-gruntwork.io-%235849a6.svg)](https://gruntwork.io/?ref=repo_cloud_nuke)

# cloud-nuke

A CLI tool to delete all resources in your cloud account. Designed for cleaning up test accounts, removing leftover resources, and eliminating unnecessary defaults like default VPCs and permissive security group rules.

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

Note that package managers are third party and may not always have the latest version. Check your version against the [releases page](https://github.com/gruntwork-io/cloud-nuke/releases).

- **macOS:** `brew install cloud-nuke`
- **Linux:** `brew install cloud-nuke` ([Homebrew on Linux](https://docs.brew.sh/Homebrew-on-Linux))
- **Windows:** `winget install cloud-nuke`

## Quick Start

```bash
# Delete all resources (with confirmation prompt)
cloud-nuke aws

# Inspect resources without deleting
cloud-nuke inspect-aws

# Delete resources in specific regions only
cloud-nuke aws --region us-east-1 --region us-west-2

# Delete only specific resource types
cloud-nuke aws --resource-type ec2 --resource-type s3

# Preview what would be deleted
cloud-nuke aws --dry-run

# Delete default VPCs and security group rules
cloud-nuke defaults-aws

# Use a config file for granular filtering
cloud-nuke aws --config path/to/config.yaml
```

## Supported Resources

Supports 125+ AWS resource types. [Full list →](docs/supported-resources.md)

## Documentation

| Topic | Description |
|---|---|
| [CLI Usage](docs/cli-usage.md) | All flags, options, and commands |
| [Configuration](docs/configuration.md) | Config file format and filter examples |
| [Supported Resources](docs/supported-resources.md) | Full resource list and config support matrix |
| [Library Usage](docs/library-usage.md) | Using cloud-nuke as a Go library |
| [Developing](docs/developing.md) | Running locally, testing, and releasing |

## Telemetry

As of `v0.29.0`, cloud-nuke sends telemetry to Gruntwork (command name, version, and AWS account ID). IP addresses and resource names are never collected. Disable with `DISABLE_TELEMETRY=1`.

## Credentials

You will need to provide your AWS credentials using one of the [standard AWS CLI credential mechanisms](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html).

## Contributing

Contributions are very welcome! Please check out the [Contribution Guidelines](CONTRIBUTING.md) and [Developing cloud-nuke](docs/developing.md) for instructions.

## License

This code is released under the MIT License. See [LICENSE.txt](/LICENSE.txt).
