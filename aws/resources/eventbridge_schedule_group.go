package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// EventBridgeScheduleGroupAPI defines the interface for EventBridge Schedule Group operations.
type EventBridgeScheduleGroupAPI interface {
	DeleteScheduleGroup(ctx context.Context, params *scheduler.DeleteScheduleGroupInput, optFns ...func(*scheduler.Options)) (*scheduler.DeleteScheduleGroupOutput, error)
	ListScheduleGroups(ctx context.Context, params *scheduler.ListScheduleGroupsInput, optFns ...func(*scheduler.Options)) (*scheduler.ListScheduleGroupsOutput, error)
}

// NewEventBridgeScheduleGroup creates a new EventBridgeScheduleGroup resource using the generic resource pattern.
func NewEventBridgeScheduleGroup() AwsResource {
	return NewAwsResource(&resource.Resource[EventBridgeScheduleGroupAPI]{
		ResourceTypeName: "event-bridge-schedule-group",
		BatchSize:        100,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EventBridgeScheduleGroupAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = scheduler.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EventBridgeScheduleGroup
		},
		Lister: listEventBridgeScheduleGroups,
		Nuker:  resource.SimpleBatchDeleter(deleteEventBridgeScheduleGroup),
	})
}

// listEventBridgeScheduleGroups retrieves all EventBridge Schedule Groups that match the config filters.
func listEventBridgeScheduleGroups(ctx context.Context, client EventBridgeScheduleGroupAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string
	paginator := scheduler.NewListScheduleGroupsPaginator(client, &scheduler.ListScheduleGroupsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[Event Bridge Schedule] Failed to list schedule groups: %s", err)
			return nil, err
		}

		for _, group := range page.ScheduleGroups {
			if aws.ToString(group.Name) == "default" {
				logging.Debug("[Event Bridge Schedule] skipping default group")
				continue
			}

			if group.State != types.ScheduleGroupStateActive {
				logging.Debugf("[Event Bridge Schedule] skipping group %s, wrong state %s", aws.ToString(group.Name), group.State)
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: group.Name,
				Time: group.CreationDate,
			}) {
				identifiers = append(identifiers, group.Name)
			}
		}
	}

	return identifiers, nil
}

// deleteEventBridgeScheduleGroup deletes a single EventBridge Schedule Group.
func deleteEventBridgeScheduleGroup(ctx context.Context, client EventBridgeScheduleGroupAPI, groupName *string) error {
	_, err := client.DeleteScheduleGroup(ctx, &scheduler.DeleteScheduleGroupInput{
		Name: groupName,
	})
	return err
}
