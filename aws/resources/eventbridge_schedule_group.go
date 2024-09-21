package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (sch *EventBridgeScheduleGroup) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("[Event Bridge Schedule] No Groups found in region %s", sch.Region)
		return nil
	}

	logging.Debugf("[Event Bridge Schedule] Deleting all Groups in %s", sch.Region)

	var deleted []*string
	for _, identifier := range identifiers {
		_, err := sch.Client.DeleteScheduleGroup(sch.Context, &scheduler.DeleteScheduleGroupInput{
			Name: identifier,
		})

		if err != nil {
			logging.Debugf(
				"[Event Bridge Schedule] Error deleting Group %s in region %s, err %s",
				*identifier,
				sch.Region,
				err,
			)
		} else {
			deleted = append(deleted, identifier)
			logging.Debugf("[Event Bridge Schedule] Deleted Group %s in region %s", *identifier, sch.Region)
		}

		e := report.Entry{
			Identifier:   aws.ToString(identifier),
			ResourceType: sch.ResourceName(),
			Error:        err,
		}
		report.Record(e)
	}

	logging.Debugf("[OK] %d Event Bridges Schedul Groups deleted in %s", len(deleted), sch.Region)
	return nil
}

func (sch *EventBridgeScheduleGroup) getAll(ctx context.Context, cnfObj config.Config) ([]*string, error) {
	var identifiers []*string
	paginator := scheduler.NewListScheduleGroupsPaginator(sch.Client, &scheduler.ListScheduleGroupsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[Event Bridge Schedule] Failed to list schedule groups: %s", err)
			return nil, errors.WithStackTrace(err)
		}

		for _, group := range page.ScheduleGroups {
			if cnfObj.EventBridgeScheduleGroup.ShouldInclude(config.ResourceValue{
				Name: group.Name,
				Time: group.CreationDate,
			}) {
				if *group.Name == "default" {
					logging.Debug("[Event Bridge Schedule] skipping default group")
					continue
				}

				if group.State != types.ScheduleGroupStateActive {
					logging.Debugf("[Event Bridge Schedule] skipping group %s, wrong state %s", *group.Name, group.State)
					continue
				}

				identifiers = append(identifiers, group.Name)
			}
		}
	}

	return identifiers, nil
}
