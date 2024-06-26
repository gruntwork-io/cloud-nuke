package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/datasync"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (dst *DataSyncTask) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("[Data Sync Task] No Data Sync Tasks found in region %s", dst.Region)
		return nil
	}

	logging.Debugf("[Data Sync Task] Deleting all Data Sync Tasks in region %s", dst.Region)
	var deleted []*string

	for _, identifier := range identifiers {
		logging.Debugf("[Data Sync Task] Deleting Data Sync Task %s in region %s", *identifier, dst.Region)
		_, err := dst.Client.DeleteTaskWithContext(dst.Context, &datasync.DeleteTaskInput{
			TaskArn: identifier,
		})
		if err != nil {
			logging.Debugf("[Data Sync Task] Error deleting Data Sync Task %s in region %s", *identifier, dst.Region)
			return err
		} else {
			deleted = append(deleted, identifier)
			logging.Debugf("[Data Sync Task] Deleted Data Sync Task %s in region %s", *identifier, dst.Region)
		}

		e := report.Entry{
			Identifier:   aws.StringValue(identifier),
			ResourceType: dst.ResourceName(),
			Error:        err,
		}
		report.Record(e)
	}

	logging.Debugf("[OK] %d Data Sync Task(s) nuked in %s", len(deleted), dst.Region)
	return nil
}

func (dst *DataSyncTask) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var identifiers []*string
	paginator := func(output *datasync.ListTasksOutput, lastPage bool) bool {
		for _, task := range output.Tasks {
			if configObj.DataSyncTask.ShouldInclude(config.ResourceValue{
				Name: task.Name,
			}) {
				identifiers = append(identifiers, task.TaskArn)
			}
		}

		return !lastPage
	}

	param := &datasync.ListTasksInput{
		MaxResults: aws.Int64(100),
	}

	if err := dst.Client.ListTasksPagesWithContext(c, param, paginator); err != nil {
		logging.Debugf("[Data Sync Task] Failed to list data sync tasks: %s", err)
		return nil, errors.WithStackTrace(err)
	}

	return identifiers, nil
}
