# Configuration

cloud-nuke uses a YAML config file for granular resource filtering. The top-level keys are resource types (use the `config key` column from the [config support matrix](supported-resources.md#config-support-matrix)). See the [examples folder](../config/examples) for more reference.

```bash
cloud-nuke aws --config path/to/file.yaml
```

## Filter Structure

Each resource type supports `include` and/or `exclude` rules:

```yaml
s3:
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

Filtering is **commutative** — include and exclude filters can be applied in any order with the same result. In the example above, buckets matching `^alb-.*-access-logs$` are included unless they also match `public` or `prod`.

### time_after / time_before

Filter resources by creation time.

```yaml
s3:
  include:
    time_after: '2020-01-01T00:00:00Z'
```

```yaml
s3:
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
s3:
  timeout: 10m
```

## Exclusion Tag

By default, resources tagged with `cloud-nuke-excluded` are excluded from deletion.
