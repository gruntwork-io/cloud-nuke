package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// CloudWatchLogGroups - represents all ec2 instances
type CloudWatchLogGroups struct {
	Client cloudwatchlogsiface.CloudWatchLogsAPI
	Region string
	Names  []string
}

// ResourceName - the simple name of the aws resource
func (r CloudWatchLogGroups) ResourceName() string {
	return "cloudwatch-loggroup"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (r CloudWatchLogGroups) ResourceIdentifiers() []string {
	return r.Names
}

func (r CloudWatchLogGroups) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that CloudWatch Logs does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We pick 35 here, which is half of what the AWS web
	// console will do. We pick a conservative number here to avoid hitting AWS API rate limits.
	return 35
}

// Nuke - nuke 'em all!!!
func (r CloudWatchLogGroups) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllCloudWatchLogGroups(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
