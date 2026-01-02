package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// EventBridgeScheduleAPI defines the interface for EventBridge Schedule operations.
type EventBridgeScheduleAPI interface {
	ListSchedules(ctx context.Context, params *scheduler.ListSchedulesInput, optFns ...func(*scheduler.Options)) (*scheduler.ListSchedulesOutput, error)
	DeleteSchedule(ctx context.Context, params *scheduler.DeleteScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.DeleteScheduleOutput, error)
}

// NewEventBridgeSchedule creates a new EventBridge Schedule resource using the generic resource pattern.
func NewEventBridgeSchedule() AwsResource {
	return NewAwsResource(&resource.Resource[EventBridgeScheduleAPI]{
		ResourceTypeName: "event-bridge-schedule",
		BatchSize:        100,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EventBridgeScheduleAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = scheduler.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EventBridgeSchedule
		},
		Lister: listEventBridgeSchedules,
		Nuker:  resource.SimpleBatchDeleter(deleteEventBridgeSchedule),
	})
}

// listEventBridgeSchedules retrieves all EventBridge schedules that match the config filters.
func listEventBridgeSchedules(ctx context.Context, client EventBridgeScheduleAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	paginator := scheduler.NewListSchedulesPaginator(client, &scheduler.ListSchedulesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
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
func deleteEventBridgeSchedule(ctx context.Context, client EventBridgeScheduleAPI, id *string) error {
	payload := strings.Split(*id, "|")
	if len(payload) != 2 {
		return fmt.Errorf("invalid identifier format: %s", *id)
	}

	_, err := client.DeleteSchedule(ctx, &scheduler.DeleteScheduleInput{
		GroupName: aws.String(payload[0]),
		Name:      aws.String(payload[1]),
	})
	return err
}
