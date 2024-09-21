package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (sch *EventBridgeSchedule) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("[Event Bridge Schedule] No Schedules found in region %s", sch.Region)
		return nil
	}

	logging.Debugf("[Event Bridge Schedule] Deleting all Schedules in %s", sch.Region)

	var deleted []*string
	for _, identifier := range identifiers {
		payload := strings.Split(*identifier, "|")
		if len(payload) != 2 {
			logging.Debugf("[Event Bridge Schedule] Invalid identifier %s", *identifier)
			continue
		}

		_, err := sch.Client.DeleteSchedule(sch.Context, &scheduler.DeleteScheduleInput{
			GroupName: aws.String(payload[0]),
			Name:      aws.String(payload[1]),
		})

		if err != nil {
			logging.Debugf(
				"[Event Bridge Schedule] Error deleting Schedule %s in region %s, err %s",
				*identifier,
				sch.Region,
				err,
			)
		} else {
			deleted = append(deleted, identifier)
			logging.Debugf("[Event Bridge Schedule] Deleted Schedule %s in region %s", *identifier, sch.Region)
		}

		e := report.Entry{
			Identifier:   aws.ToString(identifier),
			ResourceType: sch.ResourceName(),
			Error:        err,
		}
		report.Record(e)
	}

	logging.Debugf("[OK] %d Event Bridges Schedul(es) deleted in %s", len(deleted), sch.Region)
	return nil
}

func (sch *EventBridgeSchedule) getAll(ctx context.Context, cnfObj config.Config) ([]*string, error) {
	var identifiers []*string

	paginator := scheduler.NewListSchedulesPaginator(sch.Client, &scheduler.ListSchedulesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[Event Bridge Schedule] Failed to list schedules: %s", err)
			return nil, errors.WithStackTrace(err)
		}

		for _, schedule := range page.Schedules {
			id := aws.String(fmt.Sprintf("%s|%s", *schedule.GroupName, *schedule.Name))
			if cnfObj.EventBridgeSchedule.ShouldInclude(config.ResourceValue{
				Name: id,
				Time: schedule.CreationDate,
			}) {
				identifiers = append(identifiers, id)
			}
		}
	}

	return identifiers, nil
}
