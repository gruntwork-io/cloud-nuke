package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// NewEventBridgeSchedule creates a new EventBridge Schedule resource using the generic resource pattern.
func NewEventBridgeSchedule() AwsResource {
	return NewAwsResource(&resource.Resource[*scheduler.Client]{
		ResourceTypeName: "event-bridge-schedule",
		BatchSize:        100,
		InitClient: func(r *resource.Resource[*scheduler.Client], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for Scheduler client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = scheduler.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EventBridgeSchedule
		},
		Lister: listEventBridgeSchedules,
		Nuker:  resource.SimpleBatchDeleter(deleteEventBridgeSchedule),
	})
}

// listEventBridgeSchedules retrieves all EventBridge schedules that match the config filters.
func listEventBridgeSchedules(ctx context.Context, client *scheduler.Client, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	paginator := scheduler.NewListSchedulesPaginator(client, &scheduler.ListSchedulesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[Event Bridge Schedule] Failed to list schedules: %s", err)
			return nil, err
		}

		for _, schedule := range page.Schedules {
			id := aws.String(fmt.Sprintf("%s|%s", *schedule.GroupName, *schedule.Name))
			if cfg.ShouldInclude(config.ResourceValue{
				Name: id,
				Time: schedule.CreationDate,
			}) {
				identifiers = append(identifiers, id)
			}
		}
	}

	return identifiers, nil
}

// deleteEventBridgeSchedule deletes a single EventBridge schedule.
func deleteEventBridgeSchedule(ctx context.Context, client *scheduler.Client, id *string) error {
	payload := strings.Split(*id, "|")
	if len(payload) != 2 {
		logging.Debugf("[Event Bridge Schedule] Invalid identifier %s", *id)
		return fmt.Errorf("invalid identifier format: %s", *id)
	}

	_, err := client.DeleteSchedule(ctx, &scheduler.DeleteScheduleInput{
		GroupName: aws.String(payload[0]),
		Name:      aws.String(payload[1]),
	})
	if err != nil {
		return err
	}

	logging.Debugf("[Event Bridge Schedule] Deleted Schedule %s", *id)
	return nil
}
