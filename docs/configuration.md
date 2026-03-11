# Configuration

cloud-nuke uses a YAML config file for granular resource filtering. The top-level keys are resource types (use the `config key` column from the [config support matrix](supported-resources.md#config-support-matrix)). See the [examples folder](../config/examples) for more reference.

```bash
cloud-nuke aws --config path/to/file.yaml
```

## Filter Structure

Each resource type supports `include` and/or `exclude` rules:

```yaml
S3:
  include:
    names_regex:
      - ^alb-.*-access-logs$
  exclude:
    names_regex:
      - public
```

## Filters

### names_regex

Match resources by name using regular expressions.

```yaml
S3:
  include:
    names_regex:
      - ^alb-.*-access-logs$
      - .*-prod-alb-.*
  exclude:
    names_regex:
      - public
      - prod
```

Filtering is **commutative** — include and exclude filters can be applied in any order with the same result. In the example above, buckets matching `^alb-.*-access-logs$` are included unless they also match `public` or `prod`.

### time_after / time_before

Filter resources by creation time.

```yaml
S3:
  include:
    time_after: '2020-01-01T00:00:00Z'
```

```yaml
S3:
  include:
    time_before: '2020-01-01T00:00:00Z'
```

### tags

Filter resources by AWS tags. Tag names are case-sensitive; tag values are lowercased before regex matching.

```yaml
EC2:
  include:
    tags:
      Environment: "test"
      Owner: "dev-team"
    tags_operator: "OR"   # Include if ANY tag matches (default)
```

```yaml
SecurityGroup:
  exclude:
    tags:
      Team: ".*"
      Service: ".*"
    tags_operator: "AND"  # Exclude only if ALL tags present
```

The `tags_operator` field supports `AND` and `OR` (case-insensitive). Default is `OR`.

This is useful for tagging enforcement — the example above nukes resources missing either required tag while keeping properly-tagged resources safe.

### timeout

Set per-resource-type execution timeout:

```yaml
S3:
  timeout: 10m
```

### protect_until_expire

Time-based protection using the `cloud-nuke-after` tag. This feature is **enabled globally by default** — all resources with a valid `cloud-nuke-after` tag and a future timestamp are automatically protected from deletion.

To protect a resource, tag it with `cloud-nuke-after` and an RFC 3339 timestamp:

```
cloud-nuke-after = 2026-06-01T00:00:00Z
```

The resource will be excluded from deletion until after `2026-06-01T00:00:00Z`. Once the timestamp passes, the resource becomes eligible for deletion again.

> **Note:** This only works for resources that support tag-based filtering (see the `tags` column in the [config support matrix](supported-resources.md#config-support-matrix)). Resources without tag support cannot be protected this way.

### default_only

Limit operations to AWS-managed default resources (e.g., default VPC, default security group, default subnets). This applies only to EC2 resource types: `EC2Endpoint`, `EC2Subnet`, `NATGateway`, `VPC`, `InternetGateway`, `NetworkInterface`, `SecurityGroup`, and `RouteTable`.

```yaml
VPC:
  default_only: true
```

This is primarily used by the `defaults-aws` command, which sets it automatically. When set in a config file, only default resources of that type will be targeted.

### include_unaliased_keys

For KMS customer-managed keys, controls whether keys without aliases are included. By default, unaliased keys are excluded to avoid accidentally deleting keys that may be in use but unnamed. Can also be set via the `--list-unaliased-kms-keys` or `--delete-unaliased-kms-keys` CLI flags.

```yaml
KMSCustomerKeys:
  include_unaliased_keys: true
```

## Exclusion Tag

Resources tagged with `cloud-nuke-excluded = true` are excluded from deletion. The tag value must be `"true"` (case-insensitive) — an empty value or other values like `"false"` will not trigger exclusion.
