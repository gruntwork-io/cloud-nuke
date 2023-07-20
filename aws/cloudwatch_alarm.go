package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/gruntwork-io/cloud-nuke/config"
)

func getAllCloudWatchAlarms(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := cloudwatch.New(session)

	allAlarms := []*string{}
	input := &cloudwatch.DescribeAlarmsInput{
		AlarmTypes: aws.StringSlice([]string{cloudwatch.AlarmTypeMetricAlarm, cloudwatch.AlarmTypeCompositeAlarm}),
	}
	err := svc.DescribeAlarmsPages(
		input,
		func(page *cloudwatch.DescribeAlarmsOutput, lastPage bool) bool {
			for _, alarm := range page.MetricAlarms {
				if shouldIncludeCloudWatchMetricAlarm(alarm, excludeAfter, configObj) {
					allAlarms = append(allAlarms, alarm.AlarmName)
				}
			}
			for _, alarm := range page.CompositeAlarms {
				if shouldIncludeCloudWatchCompositeAlarm(alarm, excludeAfter, configObj) {
					allAlarms = append(allAlarms, alarm.AlarmName)
				}
			}
			return !lastPage
		},
	)
	return allAlarms, errors.WithStackTrace(err)
}

func shouldIncludeCloudWatchCompositeAlarm(alarm *cloudwatch.CompositeAlarm, excludeAfter time.Time, configObj config.Config) bool {
	if alarm == nil {
		return false
	}

	if alarm.AlarmConfigurationUpdatedTimestamp != nil && excludeAfter.Before(*alarm.AlarmConfigurationUpdatedTimestamp) {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(alarm.AlarmName),
		configObj.CloudWatchAlarm.IncludeRule.NamesRegExp,
		configObj.CloudWatchAlarm.ExcludeRule.NamesRegExp,
	)
}

func shouldIncludeCloudWatchMetricAlarm(alarm *cloudwatch.MetricAlarm, excludeAfter time.Time, configObj config.Config) bool {
	if alarm == nil {
		return false
	}

	if alarm.AlarmConfigurationUpdatedTimestamp != nil && excludeAfter.Before(*alarm.AlarmConfigurationUpdatedTimestamp) {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(alarm.AlarmName),
		configObj.CloudWatchAlarm.IncludeRule.NamesRegExp,
		configObj.CloudWatchAlarm.ExcludeRule.NamesRegExp,
	)
}

func nukeAllCloudWatchAlarms(session *session.Session, identifiers []*string) error {
	region := aws.StringValue(session.Config.Region)

	svc := cloudwatch.New(session)

	if len(identifiers) == 0 {
		logging.Logger.Debugf("No CloudWatch Alarms to nuke in region %s", region)
		return nil
	}

	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on CloudWatchAlarms.MaxBatchSize, however we add a guard here to warn users when the batching fails and has a
	// chance of throttling AWS. Since we concurrently make one call for each identifier, we pick 100 for the limit here
	// because many APIs in AWS have a limit of 100 requests per second.
	if len(identifiers) > 100 {
		logging.Logger.Errorf("Nuking too many CloudWatch Alarms at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyCloudWatchAlarmsErr{}
	}

	logging.Logger.Debugf("Deleting CloudWatch Alarms in region %s", region)

	// If the alarm's type is composite alarm, remove the dependency by removing the rule.
	alarms, err := svc.DescribeAlarms(&cloudwatch.DescribeAlarmsInput{
		AlarmTypes: aws.StringSlice([]string{cloudwatch.AlarmTypeMetricAlarm, cloudwatch.AlarmTypeCompositeAlarm}),
		AlarmNames: identifiers,
	})
	if err != nil {
		logging.Logger.Debugf("[Failed] %s", err)
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error Nuking Cloudwatch Alarm Dependency",
		}, map[string]interface{}{
			"region": *session.Config.Region,
		})
	}

	for _, compositeAlarm := range alarms.CompositeAlarms {
		_, err := svc.PutCompositeAlarm(&cloudwatch.PutCompositeAlarmInput{
			AlarmName: compositeAlarm.AlarmName,
			AlarmRule: aws.String("FALSE"),
		})
		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Cloudwatch Composite Alarm",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
		}
	}

	input := cloudwatch.DeleteAlarmsInput{AlarmNames: identifiers}
	_, err = svc.DeleteAlarms(&input)

	// Record status of this resource
	e := report.BatchEntry{
		Identifiers:  aws.StringValueSlice(identifiers),
		ResourceType: "CloudWatch Alarm",
		Error:        err,
	}
	report.RecordBatch(e)

	if err != nil {
		logging.Logger.Debugf("[Failed] %s", err)
		telemetry.TrackEvent(commonTelemetry.EventContext{
			EventName: "Error Nuking Cloudwatch Alarm",
		}, map[string]interface{}{
			"region": *session.Config.Region,
		})
		return errors.WithStackTrace(err)
	}

	for _, alarmName := range identifiers {
		logging.Logger.Debugf("[OK] CloudWatch Alarm %s was deleted in %s", aws.StringValue(alarmName), region)
	}
	return nil
}

// Custom errors

type TooManyCloudWatchAlarmsErr struct{}

func (err TooManyCloudWatchAlarmsErr) Error() string {
	return "Too many CloudWatch Alarms requested at once."
}
