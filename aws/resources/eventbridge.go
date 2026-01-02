package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// EventBridgeAPI defines the interface for EventBridge operations.
type EventBridgeAPI interface {
	DeleteEventBus(ctx context.Context, params *eventbridge.DeleteEventBusInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DeleteEventBusOutput, error)
	ListEventBuses(ctx context.Context, params *eventbridge.ListEventBusesInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListEventBusesOutput, error)
}

// NewEventBridge creates a new EventBridge resource using the generic resource pattern.
func NewEventBridge() AwsResource {
	return NewAwsResource(&resource.Resource[EventBridgeAPI]{
		ResourceTypeName: "event-bridge",
		BatchSize:        100,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EventBridgeAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = eventbridge.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EventBridge
		},
		Lister: listEventBuses,
		Nuker:  resource.SimpleBatchDeleter(deleteEventBus),
	})
}

// listEventBuses retrieves all EventBridge Buses that match the config filters.
// Uses manual pagination since ListEventBuses does not have an SDK paginator.
func listEventBuses(ctx context.Context, client EventBridgeAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	hasMorePages := true
	params := &eventbridge.ListEventBusesInput{}

	for hasMorePages {
		buses, err := client.ListEventBuses(ctx, params)
		if err != nil {
			logging.Debugf("[Event Bridge] Failed to list event buses: %s", err)
			return nil, err
		}

		for _, bus := range buses.EventBuses {
			// Skip the default bus in listing
			if aws.ToString(bus.Name) == "default" {
				logging.Debugf("[Event Bridge] skipping default event bus in region %s", scope.Region)
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: bus.Name,
				Time: bus.CreationTime,
			}) {
				identifiers = append(identifiers, bus.Name)
			}
		}

		params.NextToken = buses.NextToken
		hasMorePages = params.NextToken != nil
	}

	return identifiers, nil
}

// deleteEventBus deletes a single EventBridge Bus.
func deleteEventBus(ctx context.Context, client EventBridgeAPI, busName *string) error {
	// Skip the default bus in deletion (should already be filtered out in listing,
	// but this is a safety check)
	if aws.ToString(busName) == "default" {
		return nil
	}

	_, err := client.DeleteEventBus(ctx, &eventbridge.DeleteEventBusInput{
		Name: busName,
	})
	return err
}
