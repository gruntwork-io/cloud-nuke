package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/aws-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Returns a formatted string of Snapshot snapshot ids
func getAllSnapshots(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := ec2.New(session)

	params := &ec2.DescribeSnapshotsInput{
		OwnerIds: []*string{awsgo.String("self")},
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
		logging.Logger.Infof("No Snapshots to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Terminating all Snapshots in region %s", *session.Config.Region)

	for _, snapshotID := range snapshotIds {
		params := &ec2.DeleteSnapshotInput{
			SnapshotId: snapshotID,
		}

		_, err := svc.DeleteSnapshot(params)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}

		logging.Logger.Infof("Deleted Snapshot: %s", *snapshotID)
	}

	logging.Logger.Infof("[OK] %d Snapshot(s) terminated in %s", len(snapshotIds), *session.Config.Region)
	return nil
}
