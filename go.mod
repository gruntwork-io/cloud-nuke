module github.com/gruntwork-io/cloud-nuke

go 1.24.0

require (
	cloud.google.com/go/artifactregistry v1.17.1
	cloud.google.com/go/functions v1.19.7
	cloud.google.com/go/pubsub v1.49.0
	cloud.google.com/go/storage v1.50.0
	github.com/aws/aws-sdk-go-v2 v1.41.5
	github.com/aws/aws-sdk-go-v2/config v1.29.5
	github.com/aws/aws-sdk-go-v2/service/accessanalyzer v1.36.12
	github.com/aws/aws-sdk-go-v2/service/acm v1.30.17
	github.com/aws/aws-sdk-go-v2/service/acmpca v1.37.17
	github.com/aws/aws-sdk-go-v2/service/amp v1.31.0
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.28.10
	github.com/aws/aws-sdk-go-v2/service/apigatewayv2 v1.24.16
	github.com/aws/aws-sdk-go-v2/service/apprunner v1.32.14
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.51.11
	github.com/aws/aws-sdk-go-v2/service/backup v1.40.9
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.65.2
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.45.2
	github.com/aws/aws-sdk-go-v2/service/cloudtrail v1.47.3
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.43.13
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.45.11
	github.com/aws/aws-sdk-go-v2/service/codedeploy v1.29.17
	github.com/aws/aws-sdk-go-v2/service/configservice v1.51.11
	github.com/aws/aws-sdk-go-v2/service/datapipeline v1.30.18
	github.com/aws/aws-sdk-go-v2/service/datasync v1.45.3
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.39.9
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.202.3
	github.com/aws/aws-sdk-go-v2/service/ecr v1.40.2
	github.com/aws/aws-sdk-go-v2/service/ecs v1.53.12
	github.com/aws/aws-sdk-go-v2/service/efs v1.34.10
	github.com/aws/aws-sdk-go-v2/service/eks v1.57.3
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.44.11
	github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk v1.28.15
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.28.16
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.43.11
	github.com/aws/aws-sdk-go-v2/service/eventbridge v1.36.10
	github.com/aws/aws-sdk-go-v2/service/firehose v1.36.3
	github.com/aws/aws-sdk-go-v2/service/grafana v1.26.14
	github.com/aws/aws-sdk-go-v2/service/guardduty v1.52.9
	github.com/aws/aws-sdk-go-v2/service/iam v1.39.0
	github.com/aws/aws-sdk-go-v2/service/kafka v1.38.15
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.32.17
	github.com/aws/aws-sdk-go-v2/service/kms v1.37.17
	github.com/aws/aws-sdk-go-v2/service/lambda v1.69.11
	github.com/aws/aws-sdk-go-v2/service/macie2 v1.44.8
	github.com/aws/aws-sdk-go-v2/service/mq v1.34.19
	github.com/aws/aws-sdk-go-v2/service/networkfirewall v1.44.13
	github.com/aws/aws-sdk-go-v2/service/opensearch v1.45.10
	github.com/aws/aws-sdk-go-v2/service/ram v1.36.2
	github.com/aws/aws-sdk-go-v2/service/rds v1.93.11
	github.com/aws/aws-sdk-go-v2/service/redshift v1.53.11
	github.com/aws/aws-sdk-go-v2/service/route53 v1.48.6
	github.com/aws/aws-sdk-go-v2/service/s3 v1.75.3
	github.com/aws/aws-sdk-go-v2/service/s3control v1.53.3
	github.com/aws/aws-sdk-go-v2/service/sagemaker v1.174.1
	github.com/aws/aws-sdk-go-v2/service/scheduler v1.12.16
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.34.17
	github.com/aws/aws-sdk-go-v2/service/securityhub v1.55.8
	github.com/aws/aws-sdk-go-v2/service/servicediscovery v1.39.3
	github.com/aws/aws-sdk-go-v2/service/ses v1.29.9
	github.com/aws/aws-sdk-go-v2/service/sns v1.33.18
	github.com/aws/aws-sdk-go-v2/service/sqs v1.37.13
	github.com/aws/aws-sdk-go-v2/service/ssm v1.56.0
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.13
	github.com/aws/aws-sdk-go-v2/service/vpclattice v1.13.9
	github.com/aws/smithy-go v1.24.2
	github.com/go-errors/errors v1.4.2
	github.com/gruntwork-io/go-commons v0.17.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/pterm/pterm v0.12.45
	github.com/sirupsen/logrus v1.8.3
	github.com/stretchr/testify v1.11.1
	github.com/urfave/cli/v2 v2.10.3
	google.golang.org/api v0.247.0
	google.golang.org/genproto v0.0.0-20250603155806-513f23925822
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217
	google.golang.org/grpc v1.79.3
	google.golang.org/protobuf v1.36.10
	gopkg.in/yaml.v2 v2.4.0
)

require (
	atomicgo.dev/cursor v0.1.1 // indirect
	atomicgo.dev/keyboard v0.2.8 // indirect
	cel.dev/expr v0.25.1 // indirect
	cloud.google.com/go v0.120.0 // indirect
	cloud.google.com/go/auth v0.16.4 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloud.google.com/go/iam v1.5.2 // indirect
	cloud.google.com/go/longrunning v0.6.7 // indirect
	cloud.google.com/go/monitoring v1.24.2 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.30.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.50.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.50.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.8 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.58 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.27 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.2 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.31 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.5.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.10.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.24.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.28.13 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cncf/xds/go v0.0.0-20251210132809-ee656c7534f5 // indirect
	github.com/containerd/console v1.0.3 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.36.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/googleapis/gax-go/v2 v2.15.0 // indirect
	github.com/gookit/color v1.5.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/lithammer/fuzzysearch v1.1.5 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/spiffe/go-spiffe/v2 v2.6.0 // indirect
	github.com/xo/terminfo v0.0.0-20210125001918-ca9a967f8778 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.39.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.61.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.61.0 // indirect
	go.opentelemetry.io/otel v1.40.0 // indirect
	go.opentelemetry.io/otel/metric v1.40.0 // indirect
	go.opentelemetry.io/otel/sdk v1.40.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.40.0 // indirect
	go.opentelemetry.io/otel/trace v1.40.0 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/exp v0.0.0-20221106115401-f9659909a136 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/oauth2 v0.34.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/term v0.38.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251202230838-ff82c1b0f217 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
