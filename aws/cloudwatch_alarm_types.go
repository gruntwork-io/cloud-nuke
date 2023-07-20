package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// CloudWatchAlarm - represents all CloudWatchAlarm that should be deleted.
type CloudWatchAlarm struct {
	Client     cloudwatchiface.CloudWatchAPI
	Region     string
	AlarmNames []string
}

// ResourceName - the simple name of the aws resource
func (cwal CloudWatchAlarm) ResourceName() string {
	return "cloudwatch-alarm"
}

// ResourceIdentifiers - The name of cloudwatch alarms
func (cwal CloudWatchAlarm) ResourceIdentifiers() []string {
	return cwal.AlarmNames
}

func (cwal CloudWatchAlarm) MaxBatchSize() int {
	return 99
}

// Nuke - nuke 'em all!!!
func (cwal CloudWatchAlarm) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllCloudWatchAlarms(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
