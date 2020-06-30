package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

func getAllRdsSnapshots(session *session.Session, excludeAfter time.Time) ([]*string, error) {
	svc := rds.New(session)

	result, err := svc.DescribeDBSnapshots(&rds.DescribeDBSnapshotsInput{})

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var snapshots []*string

	for _, database := range result.DBSnapshots {
		if database.SnapshotCreateTime != nil && excludeAfter.After(awsgo.TimeValue(database.SnapshotCreateTime)) {
			snapshots = append(snapshots, database.DBSnapshotIdentifier)
		}
	}

	return snapshots, nil
}

func nukeAllRdsSnapshots(session *session.Session, snapshots []*string) error {
	svc := rds.New(session)

	if len(snapshots) == 0 {
		logging.Logger.Infof("No RDS Snapshot to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all RDS Snapshots in region %s", *session.Config.Region)
	deletedSnapShots := []*string{}

	for _, snapshot := range snapshots {
		params := &rds.DeleteDBSnapshotInput{
			DBSnapshotIdentifier: snapshot,
		}

		_, err := svc.DeleteDBSnapshot(params)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s: %s", *snapshot, err)
		} else {
			deletedSnapShots = append(deletedSnapShots, snapshot)
			logging.Logger.Infof("Deleted RDS Snapshot: %s", awsgo.StringValue(snapshot))
		}
	}

	if len(deletedSnapShots) > 0 {
		for _, snapshot := range deletedSnapShots {

			err := svc.WaitUntilDBSnapshotDeleted(&rds.DescribeDBSnapshotsInput{
				DBSnapshotIdentifier: snapshot,
			})

			if err != nil {
				logging.Logger.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}

	if deletedSnapShots != snapshots {
		logging.Logger.Errorf("[Failed] - %d/%d - RDS Snapshot(s) failed deletion in %s", snapshots-deletedSnapShots, snapshots, *session.Config.Region)
	}

	logging.Logger.Infof("[OK] %d RDS DB Snapshot(s) deleted in %s", len(deletedSnapShots), *session.Config.Region)
	return nil
}
