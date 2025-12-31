package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EventBridgeAPI interface {
	DeleteEventBus(ctx context.Context, params *eventbridge.DeleteEventBusInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DeleteEventBusOutput, error)
	ListEventBuses(ctx context.Context, params *eventbridge.ListEventBusesInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListEventBusesOutput, error)
}

type EventBridge struct {
	BaseAwsResource
	Client     EventBridgeAPI
	Region     string
	EventBuses []string
}

func (eb *EventBridge) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EventBridge
}

func (eb *EventBridge) Init(cfg aws.Config) {
	eb.Client = eventbridge.NewFromConfig(cfg)
}

func (eb *EventBridge) ResourceName() string { return "event-bridge" }

func (eb *EventBridge) ResourceIdentifiers() []string { return eb.EventBuses }

func (eb *EventBridge) MaxBatchSize() int {
	return 100
}

func (eb *EventBridge) Nuke(ctx context.Context, identifiers []string) error {
	if err := eb.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (eb *EventBridge) GetAndSetIdentifiers(ctx context.Context, cnfObj config.Config) ([]string, error) {
	identifiers, err := eb.getAll(ctx, cnfObj)
	if err != nil {
		return nil, err
	}

	eb.EventBuses = aws.ToStringSlice(identifiers)
	return eb.EventBuses, nil
}
