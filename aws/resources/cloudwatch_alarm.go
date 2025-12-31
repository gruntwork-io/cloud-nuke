package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/hashicorp/go-multierror"
)

// CloudWatchAlarmsAPI defines the interface for CloudWatch Alarm operations.
type CloudWatchAlarmsAPI interface {
	DescribeAlarms(ctx context.Context, params *cloudwatch.DescribeAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsOutput, error)
	DeleteAlarms(ctx context.Context, params *cloudwatch.DeleteAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DeleteAlarmsOutput, error)
	PutCompositeAlarm(ctx context.Context, params *cloudwatch.PutCompositeAlarmInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.PutCompositeAlarmOutput, error)
}

// NewCloudWatchAlarms creates a new CloudWatch Alarms resource using the generic resource pattern.
func NewCloudWatchAlarms() AwsResource {
	return NewAwsResource(&resource.Resource[CloudWatchAlarmsAPI]{
		ResourceTypeName: "cloudwatch-alarm",
		BatchSize:        99,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[CloudWatchAlarmsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = cloudwatch.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.CloudWatchAlarm
		},
		Lister: listCloudWatchAlarms,
		Nuker:  nukeCloudWatchAlarms,
	})
}

// listCloudWatchAlarms retrieves all CloudWatch alarms that match the config filters.
func listCloudWatchAlarms(ctx context.Context, client CloudWatchAlarmsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var allAlarms []*string

	paginator := cloudwatch.NewDescribeAlarmsPaginator(client, &cloudwatch.DescribeAlarmsInput{
		AlarmTypes: []types.AlarmType{types.AlarmTypeMetricAlarm, types.AlarmTypeCompositeAlarm},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, alarm := range page.MetricAlarms {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: alarm.AlarmName,
				Time: alarm.AlarmConfigurationUpdatedTimestamp,
			}) {
				allAlarms = append(allAlarms, alarm.AlarmName)
			}
		}

		for _, alarm := range page.CompositeAlarms {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: alarm.AlarmName,
				Time: alarm.AlarmConfigurationUpdatedTimestamp,
			}) {
				allAlarms = append(allAlarms, alarm.AlarmName)
			}
		}
	}

	return allAlarms, nil
}

// nukeCloudWatchAlarms handles deletion of CloudWatch alarms.
// Composite alarms require special handling (clearing dependencies), while metric alarms can be bulk deleted.
func nukeCloudWatchAlarms(ctx context.Context, client CloudWatchAlarmsAPI, scope resource.Scope, resourceType string, identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No %s to nuke in %s", resourceType, scope)
		return nil
	}

	if len(identifiers) > resource.MaxBatchSizeLimit {
		logging.Errorf("Nuking too many %s at once (%d): halting to avoid hitting rate limiting",
			resourceType, len(identifiers))
		return fmt.Errorf("too many %s requested at once (%d > %d limit)", resourceType, len(identifiers), resource.MaxBatchSizeLimit)
	}

	logging.Infof("Deleting %d %s in %s", len(identifiers), resourceType, scope)

	// Classify alarms into composite vs metric
	alarms, err := client.DescribeAlarms(ctx, &cloudwatch.DescribeAlarmsInput{
		AlarmTypes: []types.AlarmType{types.AlarmTypeMetricAlarm, types.AlarmTypeCompositeAlarm},
		AlarmNames: aws.ToStringSlice(identifiers),
	})
	if err != nil {
		return fmt.Errorf("failed to describe alarms: %w", err)
	}

	// Separate composite and metric alarms
	var compositeAlarms []*string
	compositeAlarmSet := make(map[string]bool)
	for _, alarm := range alarms.CompositeAlarms {
		compositeAlarms = append(compositeAlarms, alarm.AlarmName)
		compositeAlarmSet[aws.ToString(alarm.AlarmName)] = true
	}

	var metricAlarms []*string
	for _, id := range identifiers {
		if !compositeAlarmSet[aws.ToString(id)] {
			metricAlarms = append(metricAlarms, id)
		}
	}

	var allErrs *multierror.Error

	// Delete composite alarms first using SequentialDeleter (they need prep + individual delete)
	if len(compositeAlarms) > 0 {
		compositeDeleter := resource.SequentialDeleter(deleteCompositeAlarm)
		if err := compositeDeleter(ctx, client, scope, resourceType, compositeAlarms); err != nil {
			allErrs = multierror.Append(allErrs, err)
		}
	}

	// Delete metric alarms using BulkDeleter
	if len(metricAlarms) > 0 {
		bulkDeleter := resource.BulkDeleter(deleteMetricAlarmsBulk)
		if err := bulkDeleter(ctx, client, scope, resourceType, metricAlarms); err != nil {
			allErrs = multierror.Append(allErrs, err)
		}
	}

	return allErrs.ErrorOrNil()
}

// deleteCompositeAlarm clears dependencies and deletes a single composite alarm.
func deleteCompositeAlarm(ctx context.Context, client CloudWatchAlarmsAPI, id *string) error {
	alarmName := aws.ToString(id)

	// Clear dependencies by setting alarm rule to FALSE
	_, err := client.PutCompositeAlarm(ctx, &cloudwatch.PutCompositeAlarmInput{
		AlarmName: id,
		AlarmRule: aws.String("FALSE"),
	})
	if err != nil {
		logging.Debugf("[Warning] failed to clear composite alarm rule %s: %s", alarmName, err)
		// Continue with deletion anyway - it might still work
	}

	// Delete the alarm
	_, err = client.DeleteAlarms(ctx, &cloudwatch.DeleteAlarmsInput{
		AlarmNames: []string{alarmName},
	})
	return err
}

// deleteMetricAlarmsBulk deletes multiple metric alarms in a single API call.
func deleteMetricAlarmsBulk(ctx context.Context, client CloudWatchAlarmsAPI, ids []string) error {
	_, err := client.DeleteAlarms(ctx, &cloudwatch.DeleteAlarmsInput{
		AlarmNames: ids,
	})
	return err
}
