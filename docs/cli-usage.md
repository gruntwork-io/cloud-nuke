# CLI Usage

## Commands

| Command | Description |
|---|---|
| `cloud-nuke aws` | Delete all resources (with confirmation prompt) |
| `cloud-nuke inspect-aws` | Inspect resources without deleting |
| `cloud-nuke defaults-aws` | Delete default VPCs and default security group rules |
| `cloud-nuke azure` | Delete Azure resources (with confirmation prompt) |
| `cloud-nuke inspect-azure` | Inspect Azure resources without deleting |
| `cloud-nuke gcp` | Delete GCP resources (with confirmation prompt) |
| `cloud-nuke inspect-gcp` | Inspect GCP resources without deleting |

## Flags

### Filtering

| Flag | Description | Available in |
|---|---|---|
| `--region` | Target specific regions (repeatable). On GCP this names locations, which filter what each resource keeps, and `global` is not one of them. A multi-region or dual-region GCS bucket answers to its own location, `US` or `NAM4`, not to a region inside it. On Azure this names locations, typed the way the portal shows them or the way the API reports them | aws, inspect-aws, defaults-aws, gcp, inspect-gcp, azure, inspect-azure |
| `--exclude-region` | Exclude regions (repeatable, mutually exclusive with `--region` on AWS). On GCP this excludes locations, and a resource type with no location is still deleted. On Azure this excludes locations | aws, inspect-aws, defaults-aws, gcp, inspect-gcp, azure, inspect-azure |
| `--resource-type` | Target specific resource types (repeatable) | aws, inspect-aws, gcp, inspect-gcp, azure, inspect-azure |
| `--exclude-resource-type` | Exclude resource types (repeatable, mutually exclusive with `--resource-type`) | aws, inspect-aws, gcp, inspect-gcp, azure, inspect-azure |
| `--older-than` | Only target resources older than duration ([Go duration](https://golang.org/pkg/time/#ParseDuration)) | aws, inspect-aws, gcp, inspect-gcp, azure, inspect-azure |
| `--newer-than` | Only target resources newer than duration | aws, inspect-aws, gcp, inspect-gcp, azure, inspect-azure |
| `--config` | Path to [config file](configuration.md) for granular filtering | aws, gcp, azure |
| `--exclude-first-seen` | Exclude resources based on first-seen tag | aws, inspect-aws |

Some AWS resource types report no creation time. For several of them, such as `rds-subnet-group`, `ecs-cluster`, `opensearch-domain`, `eip` and most VPC resources, cloud-nuke writes a `cloud-nuke-first-seen` tag the first time it sees the resource, and `--older-than` and `--newer-than` measure age from that tag. Writing the tag needs the service's tagging permission, for example `rds:AddTagsToResource` for `rds-subnet-group`.

The run that writes the tag treats the resource in one of two ways:

- `rds-subnet-group`, `ecs-cluster` and `opensearch-domain` are never deleted by that run, under either filter.
- `eip`, VPC resources and Network Firewall resources take the current time as their age on that run. `--older-than` does not delete them, but `--newer-than` can delete them on that same run.

`--exclude-first-seen` turns the tag off: cloud-nuke does not read it or write it. A run that also passes `--older-than` or `--newer-than` leaves the types that use the tag alone and logs one warning per type.

Some types do not return a creation time when they are listed, so cloud-nuke reads it with one extra call for each resource. It makes this call only under `--older-than` or `--newer-than`. If the call fails, for example because the permission is missing, the resource is left alone and a warning is logged.

| Resource type | Call (IAM permission) |
|---|---|
| `kinesis-stream` | `kinesis:DescribeStreamSummary`, only for a stream that `ListStreams` returns without a summary |
| `kinesis-firehose` | `firehose:DescribeDeliveryStream` |
| `data-sync-task` | `datasync:DescribeTask` |
| `data-sync-location` | the `datasync:DescribeLocation*` call for the location type, such as `datasync:DescribeLocationS3` |
| `s3-access-point` | `s3:GetAccessPoint` |
| `s3-object-lambda-access-point` | `s3:GetAccessPointForObjectLambda` |

The remaining types without a creation time, such as `rds-parameter-group`, `elasticache-subnet-group` or `route53-hosted-zone`, cannot be filtered by age. A run with a time filter, from `--older-than` and `--newer-than` or from `time_after` and `time_before` in the config file, leaves them alone and logs one warning per type. Remove the time filter to include them.

### Execution

| Flag | Description | Available in |
|---|---|---|
| `--dry-run` | Preview deletions without executing | aws, gcp, azure |
| `--force` | Skip confirmation prompt | aws, gcp, azure, defaults-aws |
| `--timeout` | Set execution timeout (e.g., `10m`). On Azure this is a deadline over the whole run | aws, gcp, azure |
| `--sg-only` | Only delete default security group rules, not VPCs | defaults-aws |

### Output

| Flag | Description | Available in |
|---|---|---|
| `--log-level` | Log verbosity: `debug`, `info` (default), `warn`, `error`, `panic`, `fatal`, `trace`. Also settable via `LOG_LEVEL` env var. | all |
| `--output-format` | Output format: `table` (default), `json` | aws, inspect-aws, gcp, inspect-gcp, azure, inspect-azure |
| `--output-file` | Write output to file instead of stdout | aws, inspect-aws, gcp, inspect-gcp, azure, inspect-azure |
| `--list-resource-types` | List all supported resource type identifiers | aws, inspect-aws, gcp, inspect-gcp, azure, inspect-azure |

### KMS

| Flag | Description | Available in |
|---|---|---|
| `--delete-unaliased-kms-keys` | Delete KMS keys without aliases | aws |
| `--list-unaliased-kms-keys` | List KMS keys without aliases | inspect-aws |

### Azure

| Flag | Description | Available in |
|---|---|---|
| `--subscription-id` | Azure subscription ID (required, repeatable) | azure, inspect-azure |
| `--region` | Target specific locations (repeatable) | azure, inspect-azure |
| `--exclude-region` | Exclude locations (repeatable) | azure, inspect-azure |

Azure authenticates through the default credential chain, so `az login`, a service principal in the environment, or a managed identity all work. A location may be typed the way the portal shows it, `West Europe`, or the way the API reports it, `westeurope`.

Azure does not report when a resource group was created, so a run that passes `--older-than` or `--newer-than` deletes none of them and says how many it left alone. Drop the time filter to include them.

A resource group another service manages, or one a `CanNotDelete` or `ReadOnly` lock covers, is left alone: Azure refuses those deletes. A lock on the subscription covers every group in it. Reading locks needs `Microsoft.Authorization/locks/read`, and without it the scan carries on and a locked group reports the delete error instead.

Deleting a resource group deletes everything inside it, wherever those resources live, so a run narrowed by `--region` can still remove a resource in another location. The run says so when a location filter is set.

### GCP

| Flag | Description | Available in |
|---|---|---|
| `--project-id` | GCP project ID (required) | gcp, inspect-gcp |

A resource type that is in no location, such as a Pub/Sub topic, cannot be narrowed by `--region`, so a run that passes one leaves that type out and says so in a warning. Drop `--region` to include it. A run whose every `--resource-type` is in no location is refused, since it could only delete nothing. `--exclude-region` is not a narrowing, so those types are still deleted under it.

The report's region column, and the `region` key in its JSON, carry the project id on GCP where they used to carry `global`, so a script selecting GCP rows by that word finds none.

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
cloud-nuke gcp --project-id my-project-id --resource-type gcs-bucket

# Nuke Azure resources in one subscription, in one location
cloud-nuke azure --subscription-id 00000000-0000-0000-0000-000000000000 --region westeurope
```

> CLI flags override config file options. If you pass `--resource-type s3` but your config only defines rules for `ec2`, only s3 is targeted.

## Protect Resources with `cloud-nuke-after` Tag

Tag resources with `cloud-nuke-after` and an ISO 8601 date (e.g., `2024-07-09T00:00:00Z`) to protect them from deletion until that date.

## Note on Nuking VPCs

Cloud-nuke automatically removes VPC dependencies: Internet Gateways, Egress Only Internet Gateways, ENIs, VPC Endpoints, Subnets, Route Tables, Network ACLs, Security Groups, and DHCP Option Sets (dissociated only). Elastic IPs are cleaned up as a separate resource first.

All other VPC sub-resources must be cleaned up before nuking VPCs.

> VPC cleanup may not fully complete on the first run due to AWS eventual consistency. If you see `InvalidParameterValue: Network interface is currently in use.`, wait 30 minutes and retry.
