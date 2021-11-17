package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// CloudWatchLogGroup - represents all ec2 instances
type CloudWatchLogGroups struct {
	Names []string
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
	// Tentative batch size to ensure AWS doesn't throttle
	return 200
}

// Nuke - nuke 'em all!!!
func (r CloudWatchLogGroups) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllCloudWatchLogGroups(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
