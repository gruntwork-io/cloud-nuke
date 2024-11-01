module github.com/gruntwork-io/cloud-nuke

go 1.21

require (
	github.com/aws/aws-sdk-go v1.49.13
	github.com/aws/aws-sdk-go-v2 v1.32.3
	github.com/aws/aws-sdk-go-v2/config v1.28.1
	github.com/aws/aws-sdk-go-v2/credentials v1.17.42
	github.com/aws/aws-sdk-go-v2/service/accessanalyzer v1.34.3
	github.com/aws/aws-sdk-go-v2/service/acm v1.30.3
	github.com/aws/aws-sdk-go-v2/service/acmpca v1.37.4
	github.com/aws/aws-sdk-go-v2/service/amp v1.30.0
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.27.3
	github.com/aws/aws-sdk-go-v2/service/apigatewayv2 v1.24.3
	github.com/aws/aws-sdk-go-v2/service/apprunner v1.32.3
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.47.0
	github.com/aws/aws-sdk-go-v2/service/backup v1.39.4
	github.com/aws/aws-sdk-go-v2/service/cloudtrail v1.44.3
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.42.3
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.43.0
	github.com/aws/aws-sdk-go-v2/service/codedeploy v1.29.3
	github.com/aws/aws-sdk-go-v2/service/configservice v1.50.3
	github.com/aws/aws-sdk-go-v2/service/datasync v1.43.0
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.36.3
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.187.0
	github.com/aws/aws-sdk-go-v2/service/ecr v1.36.3
	github.com/aws/aws-sdk-go-v2/service/ecs v1.49.0
	github.com/aws/aws-sdk-go-v2/service/efs v1.33.3
	github.com/aws/aws-sdk-go-v2/service/eks v1.51.1
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.43.1
	github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk v1.28.3
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.28.3
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.41.0
	github.com/aws/aws-sdk-go-v2/service/eventbridge v1.35.3
	github.com/aws/aws-sdk-go-v2/service/guardduty v1.50.1
	github.com/aws/aws-sdk-go-v2/service/iam v1.37.3
	github.com/aws/aws-sdk-go-v2/service/scheduler v1.12.3
	github.com/aws/aws-sdk-go-v2/service/sns v1.33.3
	github.com/aws/aws-sdk-go-v2/service/sqs v1.36.3
	github.com/aws/aws-sdk-go-v2/service/vpclattice v1.12.3
	github.com/aws/smithy-go v1.22.0
	github.com/charmbracelet/lipgloss v0.6.0
	github.com/go-errors/errors v1.4.2
	github.com/gruntwork-io/go-commons v0.17.0
	github.com/gruntwork-io/gruntwork-cli v0.7.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/pterm/pterm v0.12.45
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.8.1
	github.com/urfave/cli/v2 v2.10.3
	gopkg.in/yaml.v2 v2.4.0
)

require (
	atomicgo.dev/cursor v0.1.1 // indirect
	atomicgo.dev/keyboard v0.2.8 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.6 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.18 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.22 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.22 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.10.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.24.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.28.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.32.3 // indirect
	github.com/containerd/console v1.0.3 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/gookit/color v1.5.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/lithammer/fuzzysearch v1.1.5 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.11.1-0.20220204035834-5ac8409525e0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/urfave/cli v1.22.12 // indirect
	github.com/xo/terminfo v0.0.0-20210125001918-ca9a967f8778 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/exp v0.0.0-20221106115401-f9659909a136 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/term v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
