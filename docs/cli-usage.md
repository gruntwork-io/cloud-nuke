# CLI Usage

## Commands

| Command | Description |
|---|---|
| `cloud-nuke aws` | Delete all resources (with confirmation prompt) |
| `cloud-nuke inspect-aws` | Inspect resources without deleting |
| `cloud-nuke defaults-aws` | Delete default VPCs and default security group rules |
| `cloud-nuke gcp` | Delete GCP resources (with confirmation prompt) |
| `cloud-nuke inspect-gcp` | Inspect GCP resources without deleting |

## Flags

### Filtering

| Flag | Description | Available in |
|---|---|---|
| `--region` | Target specific regions (repeatable) | aws, inspect-aws, defaults-aws |
| `--exclude-region` | Exclude regions (repeatable, mutually exclusive with `--region`) | aws, inspect-aws, defaults-aws |
| `--resource-type` | Target specific resource types (repeatable) | aws, inspect-aws, gcp, inspect-gcp |
| `--exclude-resource-type` | Exclude resource types (repeatable, mutually exclusive with `--resource-type`) | aws, inspect-aws, gcp, inspect-gcp |
| `--older-than` | Only target resources older than duration ([Go duration](https://golang.org/pkg/time/#ParseDuration)) | aws, inspect-aws, gcp, inspect-gcp |
| `--newer-than` | Only target resources newer than duration | aws, inspect-aws, gcp, inspect-gcp |
| `--config` | Path to [config file](configuration.md) for granular filtering | aws, gcp |
| `--exclude-first-seen` | Exclude resources based on first-seen tag | aws, inspect-aws |

### Execution

| Flag | Description | Available in |
|---|---|---|
| `--dry-run` | Preview deletions without executing | aws, gcp |
| `--force` | Skip confirmation prompt | aws, gcp, defaults-aws |
| `--timeout` | Set execution timeout (e.g., `10m`) | aws, gcp |
| `--sg-only` | Only delete default security group rules, not VPCs | defaults-aws |

### Output

| Flag | Description | Available in |
|---|---|---|
| `--log-level` | Log verbosity: `debug`, `info` (default), `warn`, `error`, `panic`, `fatal`, `trace`. Also settable via `LOG_LEVEL` env var. | all |
| `--output-format` | Output format: `table` (default), `json` | aws, inspect-aws, gcp, inspect-gcp |
| `--output-file` | Write output to file instead of stdout | aws, inspect-aws, gcp, inspect-gcp |
| `--list-resource-types` | List all supported resource type identifiers | aws, inspect-aws, gcp, inspect-gcp |

### KMS

| Flag | Description | Available in |
|---|---|---|
| `--delete-unaliased-kms-keys` | Delete KMS keys without aliases | aws |
| `--list-unaliased-kms-keys` | List KMS keys without aliases | inspect-aws |

### GCP

| Flag | Description | Available in |
|---|---|---|
| `--project-id` | GCP project ID (required) | gcp, inspect-gcp |

## Examples

```bash
# Nuke everything in specific regions
cloud-nuke aws --region us-east-1 --region us-west-2

# Nuke only EC2 and S3, skip confirmation
cloud-nuke aws --resource-type ec2 --resource-type s3 --force

# Dry run with config file
cloud-nuke aws --dry-run --config path/to/config.yaml

# Inspect with specific AWS profile
AWS_PROFILE=dev cloud-nuke inspect-aws --region us-east-1

# Nuke only default security group rules
cloud-nuke defaults-aws --sg-only

# JSON output to file
cloud-nuke inspect-aws --output-format json --output-file results.json

# Nuke GCP resources
cloud-nuke gcp --project-id my-project-id --resource-type compute-instance
```

> CLI flags override config file options. If you pass `--resource-type s3` but your config only defines rules for `ec2`, only s3 is targeted.

## Protect Resources with `cloud-nuke-after` Tag

Tag resources with `cloud-nuke-after` and an ISO 8601 date (e.g., `2024-07-09T00:00:00Z`) to protect them from deletion until that date.

## Note on Nuking VPCs

Cloud-nuke automatically removes VPC dependencies: Internet Gateways, Egress Only Internet Gateways, ENIs, VPC Endpoints, Subnets, Route Tables, Network ACLs, Security Groups, and DHCP Option Sets (dissociated only). Elastic IPs are cleaned up as a separate resource first.

All other VPC sub-resources must be cleaned up before nuking VPCs.

> VPC cleanup may not fully complete on the first run due to AWS eventual consistency. If you see `InvalidParameterValue: Network interface is currently in use.`, wait 30 minutes and retry.
