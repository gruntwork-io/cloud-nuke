package aws

import (
	"regexp"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

func getAllRdsSnapshots(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := rds.New(session)

	result, err := svc.DescribeDBSnapshots(&rds.DescribeDBSnapshotsInput{})

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var snapshots []*string

	for _, database := range result.DBSnapshots {
		if database.SnapshotCreateTime != nil && excludeAfter.After(awsgo.TimeValue(database.SnapshotCreateTime)) {
			if shouldIncludeSnapshot(*database.DBSnapshotIdentifier, configObj.RDSSnapshots.IncludeRule.NamesRE, configObj.RDSSnapshots.ExcludeRule.NamesRE) {
				snapshots = append(snapshots, database.DBSnapshotIdentifier)
			}
		}
	}

	return snapshots, nil
}

func shouldIncludeSnapshot(snapshotName string, includeNamesREList []*regexp.Regexp, excludeNamesREList []*regexp.Regexp) bool {
	shouldInclude := false

	if len(includeNamesREList) > 0 {
		// If any include rules are defined
		// And the include rule matches the snapshot, check to see if an exclude rule matches
		if includeSnapshotByREList(snapshotName, includeNamesREList) {
			shouldInclude = excludeSnapshotByREList(snapshotName, excludeNamesREList)
		}
	} else if len(excludeNamesREList) > 0 {
		// If there are no include rules defined, check to see if an exclude rule matches
		shouldInclude = excludeSnapshotByREList(snapshotName, excludeNamesREList)
	} else {
		// Ohterwise
		shouldInclude = true
	}

	return shouldInclude
}

func includeSnapshotByREList(snapshotName string, reList []*regexp.Regexp) bool {
	for _, re := range reList {
		if re.MatchString(snapshotName) {
			return true
		}
	}
	return false
}

func excludeSnapshotByREList(snapshotName string, reList []*regexp.Regexp) bool {
	for _, re := range reList {
		if re.MatchString(snapshotName) {
			return false
		}
	}

	return true
}

func nukeAllRdsSnapshots(session *session.Session, snapshots []*string) error {
	svc := rds.New(session)

	if len(snapshots) == 0 {
		logging.Logger.Infof("No RDS DB Snapshot to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all RDS DB Snapshots in region %s", *session.Config.Region)
	deletedSnapshots := []*string{}

	for _, snapshot := range snapshots {
		input := &rds.DeleteDBSnapshotInput{
			DBSnapshotIdentifier: snapshot,
		}

		_, err := svc.DeleteDBSnapshot(input)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s: %s", *snapshot, err)
		} else {
			deletedSnapshots = append(deletedSnapshots, snapshot)
			logging.Logger.Infof("Deleted RDS DB Snapshot: %s", awsgo.StringValue(snapshot))
		}
	}

	if len(deletedSnapshots) > 0 {
		for _, snapshot := range deletedSnapshots {

			err := svc.WaitUntilDBSnapshotDeleted(&rds.DescribeDBSnapshotsInput{
				DBSnapshotIdentifier: snapshot,
			})

			if err != nil {
				logging.Logger.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}

	if len(deletedSnapshots) != len(snapshots) {
		logging.Logger.Errorf("[Failed] - %d/%d - RDS DB Snapshot(s) failed deletion in %s", len(snapshots)-len(deletedSnapshots), snapshots, *session.Config.Region)
	}

	logging.Logger.Infof("[OK] %d RDS DB Snapshot(s) deleted in %s", len(deletedSnapshots), *session.Config.Region)
	return nil
}
