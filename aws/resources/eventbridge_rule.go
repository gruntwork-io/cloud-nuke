package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// EventBridgeRuleAPI defines the interface for EventBridge Rule operations.
type EventBridgeRuleAPI interface {
	ListRules(ctx context.Context, params *eventbridge.ListRulesInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListRulesOutput, error)
	DeleteRule(ctx context.Context, params *eventbridge.DeleteRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DeleteRuleOutput, error)
	ListEventBuses(ctx context.Context, params *eventbridge.ListEventBusesInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListEventBusesOutput, error)
	ListTargetsByRule(ctx context.Context, params *eventbridge.ListTargetsByRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListTargetsByRuleOutput, error)
	RemoveTargets(ctx context.Context, params *eventbridge.RemoveTargetsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.RemoveTargetsOutput, error)
}

// NewEventBridgeRule creates a new EventBridgeRule resource using the generic resource pattern.
func NewEventBridgeRule() AwsResource {
	return NewAwsResource(&resource.Resource[EventBridgeRuleAPI]{
		ResourceTypeName: "event-bridge-rule",
		BatchSize:        100,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EventBridgeRuleAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = eventbridge.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EventBridgeRule
		},
		Lister: listEventBridgeRules,
		Nuker:  resource.SequentialDeleter(deleteEventBridgeRule),
	})
}

// listEventBridgeRules retrieves all EventBridge Rules that match the config filters.
// Returns identifiers in "busName|ruleName" format.
func listEventBridgeRules(ctx context.Context, client EventBridgeRuleAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	// First, get all event bus names
	busNames, err := listEventBusNames(ctx, client)
	if err != nil {
		logging.Debugf("[Event Bridge Rule] Failed to list event buses: %s", err)
		return nil, err
	}

	var identifiers []*string
	for _, busName := range busNames {
		// Manual pagination for ListRules (no SDK paginator available)
		hasMorePages := true
		params := &eventbridge.ListRulesInput{
			EventBusName: busName,
		}

		for hasMorePages {
			rules, err := client.ListRules(ctx, params)
			if err != nil {
				logging.Debugf("[Event Bridge Rule] Failed to list rules for bus %s: %s", aws.ToString(busName), err)
				return nil, err
			}

			for _, rule := range rules.Rules {
				// Create composite identifier: "busName|ruleName"
				id := aws.String(fmt.Sprintf("%s|%s", aws.ToString(busName), aws.ToString(rule.Name)))
				if cfg.ShouldInclude(config.ResourceValue{
					Name: id,
				}) {
					identifiers = append(identifiers, id)
				}
			}

			params.NextToken = rules.NextToken
			hasMorePages = params.NextToken != nil
		}
	}

	return identifiers, nil
}

// listEventBusNames retrieves all event bus names (manual pagination - no SDK paginator available).
func listEventBusNames(ctx context.Context, client EventBridgeRuleAPI) ([]*string, error) {
	var busNames []*string

	hasMorePages := true
	params := &eventbridge.ListEventBusesInput{}

	for hasMorePages {
		buses, err := client.ListEventBuses(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, bus := range buses.EventBuses {
			busNames = append(busNames, bus.Name)
		}

		params.NextToken = buses.NextToken
		hasMorePages = params.NextToken != nil
	}

	return busNames, nil
}

// deleteEventBridgeRule deletes a single EventBridge Rule.
// The identifier format is "busName|ruleName".
// It first removes all targets from the rule, then deletes the rule.
func deleteEventBridgeRule(ctx context.Context, client EventBridgeRuleAPI, identifier *string) error {
	// Parse the composite identifier
	parts := strings.Split(aws.ToString(identifier), "|")
	if len(parts) != 2 {
		return fmt.Errorf("invalid identifier format %q, expected 'busName|ruleName'", aws.ToString(identifier))
	}
	busName := parts[0]
	ruleName := parts[1]

	// Step 1: Get all targets for this rule
	targetIds, err := listTargetsByRule(ctx, client, busName, ruleName)
	if err != nil {
		logging.Debugf("[Event Bridge Rule] Error listing targets for rule %s on bus %s: %s", ruleName, busName, err)
		return err
	}

	// Step 2: Remove all targets if any exist
	if len(targetIds) > 0 {
		_, err := client.RemoveTargets(ctx, &eventbridge.RemoveTargetsInput{
			Ids:          targetIds,
			EventBusName: aws.String(busName),
			Rule:         aws.String(ruleName),
			Force:        true,
		})
		if err != nil {
			logging.Debugf("[Event Bridge Rule] Error removing targets for rule %s on bus %s: %s", ruleName, busName, err)
			return err
		}
	}

	// Step 3: Delete the rule
	_, err = client.DeleteRule(ctx, &eventbridge.DeleteRuleInput{
		EventBusName: aws.String(busName),
		Name:         aws.String(ruleName),
		Force:        true,
	})
	if err != nil {
		return err
	}

	logging.Debugf("[Event Bridge Rule] Deleted rule %s on bus %s", ruleName, busName)
	return nil
}

// listTargetsByRule retrieves all target IDs for a given rule (manual pagination).
func listTargetsByRule(ctx context.Context, client EventBridgeRuleAPI, busName, ruleName string) ([]string, error) {
	var targetIds []string

	hasMorePages := true
	params := &eventbridge.ListTargetsByRuleInput{
		Rule:         aws.String(ruleName),
		EventBusName: aws.String(busName),
	}

	for hasMorePages {
		targets, err := client.ListTargetsByRule(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, target := range targets.Targets {
			targetIds = append(targetIds, aws.ToString(target.Id))
		}

		params.NextToken = targets.NextToken
		hasMorePages = params.NextToken != nil
	}

	return targetIds, nil
}
