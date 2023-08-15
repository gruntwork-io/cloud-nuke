package resources

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// CloudWatchLogGroup - represents all ec2 instances
type CloudWatchLogGroups struct {
	Client cloudwatchlogsiface.CloudWatchLogsAPI
	Region string
	Names  []string
}

func (csr *CloudWatchLogGroups) Init(session *session.Session) {
	csr.Client = cloudwatchlogs.New(session)
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

func (csr *CloudWatchLogGroups) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := csr.getAll(configObj)
	if err != nil {
		return nil, err
	}

	csr.Names = awsgo.StringValueSlice(identifiers)
	return csr.Names, nil
}

// Nuke - nuke 'em all!!!
func (csr *CloudWatchLogGroups) Nuke(identifiers []string) error {
	if err := csr.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
