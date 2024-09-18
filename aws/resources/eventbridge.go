package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (eb *EventBridge) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("[Event Bridge] No Event Bridges found in region %s", eb.Region)
		return nil
	}

	logging.Debugf("[Event Bridge] Deleting all Event Bridges in %s", eb.Region)
	var deleted []*string

	for _, identifier := range identifiers {
		if *identifier == "default" {
			logging.Debugf("[Event Bridge] skipping deleteing default Bus in region %s", eb.Region)
			continue
		}

		_, err := eb.Client.DeleteEventBus(eb.Context, &eventbridge.DeleteEventBusInput{
			Name: identifier,
		})
		if err != nil {
			logging.Debugf(
				"[Event Bridge] Error deleting Bus %s in region %s, err %s",
				*identifier,
				eb.Region,
				err,
			)
		} else {
			deleted = append(deleted, identifier)
			logging.Debugf("[Event Bridge] Deleted Bus %s in region %s", *identifier, eb.Region)
		}

		e := report.Entry{
			Identifier:   aws.ToString(identifier),
			ResourceType: eb.ResourceName(),
			Error:        err,
		}
		report.Record(e)
	}

	logging.Debugf("[OK] %d Event Bridges Bus(es) deleted in %s", len(deleted), eb.Region)
	return nil
}

func (eb *EventBridge) getAll(ctx context.Context, cnfObj config.Config) ([]*string, error) {
	var identifiers []*string

	hasMorePages := true
	params := &eventbridge.ListEventBusesInput{}

	for hasMorePages {
		buses, err := eb.Client.ListEventBuses(ctx, params)
		if err != nil {
			logging.Debugf("[Event Bridge] Failed to list event buses: %s", err)
			return nil, errors.WithStackTrace(err)
		}

		for _, bus := range buses.EventBuses {
			if *bus.Name == "default" {
				logging.Debugf("[Event Bridge] skipping default event bus in region %s", eb.Region)
				continue
			}

			if cnfObj.EventBridge.ShouldInclude(config.ResourceValue{
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
