package aws

import (
	"time"

	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of Snapshot snapshot ids
func getAllSnapshots(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := ec2.New(session)

	// status - The status of the snapshot (pending | completed | error).
	// Since the output of this function is used to delete the returned snapshots
	// We only want to list EBS Snapshots with a status of "completed"
	// Since that is the only status that is eligible for deletion
	status_filter := ec2.Filter{Name: awsgo.String("status"), Values: []*string{awsgo.String("completed"), awsgo.String("error")}}

	params := &ec2.DescribeSnapshotsInput{
		OwnerIds: []*string{awsgo.String("self")},
		Filters:  []*ec2.Filter{&status_filter},
	}

	output, err := svc.DescribeSnapshots(params)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var snapshotIds []*string
	for _, snapshot := range output.Snapshots {
		if excludeAfter.After(*snapshot.StartTime) {
			snapshotIds = append(snapshotIds, snapshot.SnapshotId)
		}
	}

	return snapshotIds, nil
}

// Deletes all Snapshots
func nukeAllSnapshots(session *session.Session, snapshotIds []*string) error {
	svc := ec2.New(session)

	if len(snapshotIds) == 0 {
		logging.Logger.Debugf("No Snapshots to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all Snapshots in region %s", *session.Config.Region)
	var deletedSnapshotIDs []*string

	for _, snapshotID := range snapshotIds {
		params := &ec2.DeleteSnapshotInput{
			SnapshotId: snapshotID,
		}

		_, err := svc.DeleteSnapshot(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(snapshotID),
			ResourceType: "EBS Snapshot",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking EBS Snapshot",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
		} else {
			deletedSnapshotIDs = append(deletedSnapshotIDs, snapshotID)
			logging.Logger.Debugf("Deleted Snapshot: %s", *snapshotID)
		}
	}

	logging.Logger.Debugf("[OK] %d Snapshot(s) terminated in %s", len(deletedSnapshotIDs), *session.Config.Region)
	return nil
}
