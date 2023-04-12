package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// CloudWatchAlarms - represents all CloudWatchAlarms that should be deleted.
type CloudWatchAlarms struct {
	AlarmNames []string
}

// ResourceName - the simple name of the aws resource
func (cwal CloudWatchAlarms) ResourceName() string {
	return "cloudwatch-alarm"
}

// ResourceIdentifiers - The name of cloudwatch alarms
func (cwal CloudWatchAlarms) ResourceIdentifiers() []string {
	return cwal.AlarmNames
}

func (cwal CloudWatchAlarms) MaxBatchSize() int {
	return 99
}

// Nuke - nuke 'em all!!!
func (cwal CloudWatchAlarms) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllCloudWatchAlarms(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
