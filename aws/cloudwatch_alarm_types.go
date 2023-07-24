package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// CloudWatchAlarms - represents all CloudWatchAlarms that should be deleted.
type CloudWatchAlarms struct {
	Client     cloudwatchiface.CloudWatchAPI
	Region     string
	AlarmNames []string
}

// ResourceName - the simple name of the aws resource
func (cw CloudWatchAlarms) ResourceName() string {
	return "cloudwatch-alarm"
}

// ResourceIdentifiers - The name of cloudwatch alarms
func (cw CloudWatchAlarms) ResourceIdentifiers() []string {
	return cw.AlarmNames
}

func (cw CloudWatchAlarms) MaxBatchSize() int {
	return 99
}

// Nuke - nuke 'em all!!!
func (cw CloudWatchAlarms) Nuke(session *session.Session, identifiers []string) error {
	if err := cw.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
