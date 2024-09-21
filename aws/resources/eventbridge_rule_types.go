package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EventBridgeRuleAPI interface {
	ListRules(ctx context.Context, params *eventbridge.ListRulesInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListRulesOutput, error)
	DeleteRule(ctx context.Context, params *eventbridge.DeleteRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DeleteRuleOutput, error)
	ListEventBuses(ctx context.Context, params *eventbridge.ListEventBusesInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListEventBusesOutput, error)
	ListTargetsByRule(ctx context.Context, params *eventbridge.ListTargetsByRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListTargetsByRuleOutput, error)
	RemoveTargets(ctx context.Context, params *eventbridge.RemoveTargetsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.RemoveTargetsOutput, error)
}

type EventBridgeRule struct {
	BaseAwsResource
	Client EventBridgeRuleAPI
	Region string
	Rules  []string
}

func (ebr *EventBridgeRule) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EventBridgeRule
}

func (ebr *EventBridgeRule) InitV2(cfg aws.Config) {
	ebr.Client = eventbridge.NewFromConfig(cfg)
}

func (ebr *EventBridgeRule) IsUsingV2() bool { return true }

func (ebr *EventBridgeRule) ResourceName() string { return "event-bridge-rule" }

func (ebr *EventBridgeRule) ResourceIdentifiers() []string { return ebr.Rules }

func (ebr *EventBridgeRule) MaxBatchSize() int {
	return 100
}

func (ebr *EventBridgeRule) Nuke(identifiers []string) error {
	if err := ebr.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (ebr *EventBridgeRule) GetAndSetIdentifiers(ctx context.Context, cnfObj config.Config) ([]string, error) {
	identifiers, err := ebr.getAll(ctx, cnfObj)
	if err != nil {
		return nil, err
	}

	ebr.Rules = aws.ToStringSlice(identifiers)
	return ebr.Rules, nil
}
