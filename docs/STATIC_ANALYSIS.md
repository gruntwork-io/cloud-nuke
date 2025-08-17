# Static Analysis

This project uses **golangci-lint** to prevent nil pointer dereferences and other common Go bugs.

## Key Linters for Nil Safety
- **staticcheck**: Comprehensive analysis including nil checks (SA5011)
- **errcheck**: Ensures errors are checked before using returns
- **nilnil**: Prevents returning nil error with nil value
- **nilerr**: Catches returning nil when error is not nil

## Local Development

```bash
# Install and run pre-commit hooks (recommended)
pip install pre-commit
pre-commit install
pre-commit run --all-files

# Or run golangci-lint directly
golangci-lint run --new-from-rev=origin/master ./...
```

## CI/CD

CircleCI runs linting on all PRs, checking only new/modified code (`--new-from-rev=origin/master`).

## Common Fixes

### AWS SDK nil responses
```go
// ❌ Bad
output, err := client.DescribeResourcePolicy(ctx, input)
if err != nil {
    return err
}
policy := output.Policy // Potential nil panic

// ✅ Good
output, err := client.DescribeResourcePolicy(ctx, input)
if err != nil {
    return err
}
if output != nil && output.Policy != nil {
    policy := output.Policy
}
```

## Configuration

See `.golangci.yml`. Currently configured to check only new code for easier adoption.