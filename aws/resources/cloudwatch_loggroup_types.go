package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type CloudWatchLogGroupsAPI interface {
	DescribeLogGroups(ctx context.Context, params *cloudwatchlogs.DescribeLogGroupsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error)
	DeleteLogGroup(ctx context.Context, params *cloudwatchlogs.DeleteLogGroupInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DeleteLogGroupOutput, error)
}

// CloudWatchLogGroups - represents all Cloud Watch Log Groups
type CloudWatchLogGroups struct {
	BaseAwsResource
	Client CloudWatchLogGroupsAPI
	Region string
	Names  []string
}

func (csr *CloudWatchLogGroups) Init(cfg aws.Config) {
	csr.Client = cloudwatchlogs.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (csr *CloudWatchLogGroups) ResourceName() string {
	return "cloudwatch-loggroup"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (csr *CloudWatchLogGroups) ResourceIdentifiers() []string {
	return csr.Names
}

func (csr *CloudWatchLogGroups) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that CloudWatch Logs does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We pick 35 here, which is half of what the AWS web
	// console will do. We pick a conservative number here to avoid hitting AWS API rate limits.
	return 35
}

func (csr *CloudWatchLogGroups) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.CloudWatchLogGroup
}

func (csr *CloudWatchLogGroups) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := csr.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	csr.Names = aws.ToStringSlice(identifiers)
	return csr.Names, nil
}

// Nuke - nuke 'em all!!!
func (csr *CloudWatchLogGroups) Nuke(identifiers []string) error {
	if err := csr.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
