package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (ebr *EventBridgeRule) getTargets(busName, rule string) ([]string, error) {
	var identifiers []*string

	params := &eventbridge.ListTargetsByRuleInput{
		Rule:         aws.String(rule),
		EventBusName: aws.String(busName),
		NextToken:    nil,
	}
	hasMorePages := true

	for hasMorePages {
		targets, err := ebr.Client.ListTargetsByRule(ebr.Context, params)
		if err != nil {
			return nil, err
		}

		for _, target := range targets.Targets {
			identifiers = append(identifiers, target.Id)
		}

		params.NextToken = targets.NextToken
		hasMorePages = params.NextToken != nil
	}

	return aws.ToStringSlice(identifiers), nil
}

func (ebr *EventBridgeRule) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("[Event Bridge Rule] No Event Bridge Rule(s) found in region %s", ebr.Region)
		return nil
	}

	logging.Debugf("[Event Bridge Rule] Deleting all Event Bridge Rule(s) in %s", ebr.Region)
	var deleted []*string

	for _, identifier := range identifiers {
		payload := strings.Split(*identifier, "|")
		if len(payload) != 2 {
			logging.Debugf("[Event Bridge Rule] Invalid identifier %s", *identifier)
			continue
		}

		ids, errGetTarget := ebr.getTargets(payload[0], payload[1])
		if errGetTarget != nil {
			logging.Debugf("[Event Bridge Rule] error when listing targets for rule %s, %s", payload[1], errGetTarget)
			return errGetTarget
		}

		if len(ids) != 0 {
			_, errRmTargets := ebr.Client.RemoveTargets(ebr.Context, &eventbridge.RemoveTargetsInput{
				Ids:          ids,
				EventBusName: aws.String(payload[0]),
				Rule:         aws.String(payload[1]),
				Force:        true,
			})
			if errRmTargets != nil {
				logging.Debugf("[Event Bridge Rule] error when removing rule %s, targets %s", payload[1], errRmTargets)
				return errRmTargets
			}
		}

		_, err := ebr.Client.DeleteRule(ebr.Context, &eventbridge.DeleteRuleInput{
			EventBusName: aws.String(payload[0]),
			Name:         aws.String(payload[1]),
			Force:        true,
		})
		if err != nil {
			logging.Debugf(
				"[Event Bridge Rule] Error deleting Rule %s in region %s, bus %s, err %s",
				payload[1],
				ebr.Region,
				payload[0],
				err,
			)
		} else {
			deleted = append(deleted, identifier)
			logging.Debugf(
				"[Event Bridge Rule] Deleted Rule %s in region %s, bus %s",
				payload[1],
				ebr.Region,
				payload[0],
			)
		}

		e := report.Entry{
			Identifier:   aws.ToString(identifier),
			ResourceType: ebr.ResourceName(),
			Error:        err,
		}
		report.Record(e)
	}

	logging.Debugf("[OK] %d Event Bridge Rule(s) deleted in %s", len(deleted), ebr.Region)
	return nil
}

func (ebr *EventBridgeRule) getBusNames() ([]*string, error) {
	var identifiers []*string

	hasMorePages := true
	params := &eventbridge.ListEventBusesInput{}

	for hasMorePages {
		buses, err := ebr.Client.ListEventBuses(ebr.Context, params)
		if err != nil {
			return nil, err
		}

		for _, bus := range buses.EventBuses {
			identifiers = append(identifiers, bus.Name)
		}

		params.NextToken = buses.NextToken
		hasMorePages = params.NextToken != nil
	}

	return identifiers, nil
}

func (ebr *EventBridgeRule) getAll(ctx context.Context, cnfObj config.Config) ([]*string, error) {
	var identifiers []*string

	eBusNames, errGetBus := ebr.getBusNames()
	if errGetBus != nil {
		logging.Debugf("[Event Bridge] Failed to list event buses: %s", errGetBus)
		return nil, errors.WithStackTrace(errGetBus)
	}

	for _, bus := range eBusNames {
		hasMorePages := true
		params := &eventbridge.ListRulesInput{
			EventBusName: bus,
		}

		for hasMorePages {
			rules, err := ebr.Client.ListRules(ctx, params)
			if err != nil {
				logging.Debugf("[Event Bridge] Failed to list event rules: %s", err)
				return nil, errors.WithStackTrace(err)
			}

			for _, rule := range rules.Rules {
				id := aws.String(fmt.Sprintf("%s|%s", *bus, *rule.Name))
				if cnfObj.EventBridgeRule.ShouldInclude(config.ResourceValue{
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
