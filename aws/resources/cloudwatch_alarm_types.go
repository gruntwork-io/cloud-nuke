package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type CloudWatchAlarmsAPI interface {
	DescribeAlarms(ctx context.Context, params *cloudwatch.DescribeAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsOutput, error)
	DeleteAlarms(ctx context.Context, params *cloudwatch.DeleteAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DeleteAlarmsOutput, error)
	PutCompositeAlarm(ctx context.Context, params *cloudwatch.PutCompositeAlarmInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.PutCompositeAlarmOutput, error)
}

// CloudWatchAlarms - represents all CloudWatchAlarms that should be deleted.
type CloudWatchAlarms struct {
	BaseAwsResource
	Client     CloudWatchAlarmsAPI
	Region     string
	AlarmNames []string
}

func (cw *CloudWatchAlarms) InitV2(cfg aws.Config) {
	cw.Client = cloudwatch.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (cw *CloudWatchAlarms) ResourceName() string {
	return "cloudwatch-alarm"
}

// ResourceIdentifiers - The name of cloudwatch alarms
func (cw *CloudWatchAlarms) ResourceIdentifiers() []string {
	return cw.AlarmNames
}

func (cw *CloudWatchAlarms) MaxBatchSize() int {
	return 99
}

func (cw *CloudWatchAlarms) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.CloudWatchAlarm
}

func (cw *CloudWatchAlarms) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := cw.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	cw.AlarmNames = aws.ToStringSlice(identifiers)
	return cw.AlarmNames, nil
}

// Nuke - nuke 'em all!!!
func (cw *CloudWatchAlarms) Nuke(identifiers []string) error {
	if err := cw.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
