# Developing cloud-nuke

## Running Locally

```bash
go run main.go
```

## Running Tests

> **WARNING:** Many tests run against a real AWS account and create/destroy actual resources. DO NOT hit `CTRL+C` while tests are running — this prevents proper cleanup. We are not responsible for any charges you may incur.

Configure your [AWS credentials](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html) before running tests.

```bash
# Run all tests
go test -v ./...

# Run tests in a specific package
cd aws
go test -v

# Run a specific test
cd aws
go test -v -run TestListAMIs

# Run a specific test in aws/resources
cd aws/resources
go test -v -run TestLambdaFunction_GetAll

# Opt-in to expensive tests via env vars
TEST_ACMPCA_EXPENSIVE_ENABLE=1 go test -v ./...
```

## Formatting

Every source file should be formatted with `go fmt`.

## Releasing New Versions

We follow the release process defined in our [Coding Methodology](https://www.notion.so/gruntwork/Gruntwork-Coding-Methodology-02fdcd6e4b004e818553684760bf691e#08b68ee0e19143e89523dcf483d2bf48).

### Choosing a Release Tag

If the release includes new resource types, bump the **minor version** (`X` in `v0.X.Y`). Since `v0.2.0`, cloud-nuke automatically includes new resources (opt-out model), so new resource types are inherently backward-incompatible for users with CI pipelines around cloud-nuke.

### Creating a Release

Go to the [Releases Page](https://github.com/gruntwork-io/cloud-nuke/releases) and create a new release. CircleCI will automatically detect the tag, build binaries for every OS, and upload them to the GitHub release.
