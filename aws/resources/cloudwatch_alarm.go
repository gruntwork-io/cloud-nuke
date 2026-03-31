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
	"github.com/gruntwork-io/cloud-nuke/util"
)

// CloudWatchAlarmsAPI defines the interface for CloudWatch Alarm operations.
type CloudWatchAlarmsAPI interface {
	DescribeAlarms(ctx context.Context, params *cloudwatch.DescribeAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsOutput, error)
	DeleteAlarms(ctx context.Context, params *cloudwatch.DeleteAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DeleteAlarmsOutput, error)
	PutCompositeAlarm(ctx context.Context, params *cloudwatch.PutCompositeAlarmInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.PutCompositeAlarmOutput, error)
	ListTagsForResource(ctx context.Context, params *cloudwatch.ListTagsForResourceInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.ListTagsForResourceOutput, error)
}

// NewCloudWatchAlarms creates a new CloudWatch Alarms resource using the generic resource pattern.
func NewCloudWatchAlarms() AwsResource {
	return NewAwsResource(&resource.Resource[CloudWatchAlarmsAPI]{
		ResourceTypeName: "cloudwatch-alarm",
		// CloudWatch DeleteAlarms API accepts a maximum of 100 alarm names per call.
		BatchSize: 99,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[CloudWatchAlarmsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = cloudwatch.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.CloudWatchAlarm
		},
		Lister: listCloudWatchAlarms,
		Nuker:  resource.SequentialDeleter(deleteCloudWatchAlarm),
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
			tagsOutput, err := client.ListTagsForResource(ctx, &cloudwatch.ListTagsForResourceInput{
				ResourceARN: alarm.AlarmArn,
			})
			if err != nil {
				logging.Debugf("[cloudwatch-alarm] Failed to list tags for alarm %s: %s", aws.ToString(alarm.AlarmName), err)
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: alarm.AlarmName,
				Time: alarm.AlarmConfigurationUpdatedTimestamp,
				Tags: util.ConvertCloudWatchTagsToMap(tagsOutput.Tags),
			}) {
				allAlarms = append(allAlarms, alarm.AlarmName)
			}
		}

		for _, alarm := range page.CompositeAlarms {
			tagsOutput, err := client.ListTagsForResource(ctx, &cloudwatch.ListTagsForResourceInput{
				ResourceARN: alarm.AlarmArn,
			})
			if err != nil {
				logging.Debugf("[cloudwatch-alarm] Failed to list tags for alarm %s: %s", aws.ToString(alarm.AlarmName), err)
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: alarm.AlarmName,
				Time: alarm.AlarmConfigurationUpdatedTimestamp,
				Tags: util.ConvertCloudWatchTagsToMap(tagsOutput.Tags),
			}) {
				allAlarms = append(allAlarms, alarm.AlarmName)
			}
		}
	}

	return allAlarms, nil
}

// deleteCloudWatchAlarm deletes a single CloudWatch alarm.
// For composite alarms, it first clears dependencies by setting the rule to FALSE.
func deleteCloudWatchAlarm(ctx context.Context, client CloudWatchAlarmsAPI, id *string) error {
	alarmName := aws.ToString(id)

	// Check if this is a composite alarm that needs dependency clearing
	alarms, err := client.DescribeAlarms(ctx, &cloudwatch.DescribeAlarmsInput{
		AlarmTypes: []types.AlarmType{types.AlarmTypeCompositeAlarm},
		AlarmNames: []string{alarmName},
	})
	if err != nil {
		return fmt.Errorf("failed to describe alarm %s: %w", alarmName, err)
	}

	// If it's a composite alarm, clear dependencies first
	if len(alarms.CompositeAlarms) > 0 {
		_, err = client.PutCompositeAlarm(ctx, &cloudwatch.PutCompositeAlarmInput{
			AlarmName: id,
			AlarmRule: aws.String("FALSE"),
		})
		if err != nil {
			logging.Debugf("[Warning] failed to clear composite alarm rule %s: %s", alarmName, err)
			// Continue with deletion anyway - it might still work
		}
	}

	// Delete the alarm
	_, err = client.DeleteAlarms(ctx, &cloudwatch.DeleteAlarmsInput{
		AlarmNames: []string{alarmName},
	})
	return err
}
