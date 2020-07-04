package aws

import (
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Built-in waiter function WaitUntilDBClusterSnapshotDeleted not working as expected.
// Created a custom one
func waitUntilRdsClusterSnapshotDeleted(svc *rds.RDS, input *rds.DescribeDBClusterSnapshotsInput) error {
	for i := 0; i < 90; i++ {
		_, err := svc.DescribeDBClusterSnapshots(input)
		if err != nil {
			if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == rds.ErrCodeDBClusterSnapshotNotFoundFault {
				return nil
			}

			return err
		}

		time.Sleep(10 * time.Second)
		logging.Logger.Debug("Waiting for RDS DB Cluster snapshot to be deleted...")
	}

	return RdsClusterSnapshotDeleteError{name: *input.DBClusterSnapshotIdentifier}
}

func getAllRdsClusterSnapshots(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := rds.New(session)

	result, err := svc.DescribeDBClusterSnapshots(&rds.DescribeDBClusterSnapshotsInput{})

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var snapshots []*string

	for _, database := range result.DBClusterSnapshots {
		if database.SnapshotCreateTime != nil && excludeAfter.After(awsgo.TimeValue(database.SnapshotCreateTime)) {
			if shouldIncludeClusterSnapshot(*database.DBClusterSnapshotIdentifier, configObj.RDSSnapshots.IncludeRule.NamesRE, configObj.RDSSnapshots.ExcludeRule.NamesRE) {
				snapshots = append(snapshots, database.DBClusterSnapshotIdentifier)
			}
		}
	}

	return snapshots, nil
}

func shouldIncludeClusterSnapshot(snapshotName string, includeNamesREList []*regexp.Regexp, excludeNamesREList []*regexp.Regexp) bool {
	shouldInclude := false

	if len(includeNamesREList) > 0 {
		// If any include rules are defined
		// And the include rule matches the snapshot, check to see if an exclude rule matches
		if includeClusterSnapshotByREList(snapshotName, includeNamesREList) {
			shouldInclude = excludeClusterSnapshotByREList(snapshotName, excludeNamesREList)
		}
	} else if len(excludeNamesREList) > 0 {
		// If there are no include rules defined, check to see if an exclude rule matches
		shouldInclude = excludeClusterSnapshotByREList(snapshotName, excludeNamesREList)
	} else {
		// Otherwise
		shouldInclude = true
	}

	return shouldInclude
}

func includeClusterSnapshotByREList(snapshotName string, reList []*regexp.Regexp) bool {
	for _, re := range reList {
		if re.MatchString(snapshotName) {
			return true
		}
	}
	return false
}

func excludeClusterSnapshotByREList(snapshotName string, reList []*regexp.Regexp) bool {
	for _, re := range reList {
		if re.MatchString(snapshotName) {
			return false
		}
	}

	return true
}

func nukeAllRdsClusterSnapshots(session *session.Session, snapshots []*string) error {
	svc := rds.New(session)

	if len(snapshots) == 0 {
		logging.Logger.Infof("No RDS DB Cluster Snapshot to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all RDS DB Cluster Snapshots in region %s", *session.Config.Region)
	deletedSnapshots := []*string{}

	for _, snapshot := range snapshots {
		input := &rds.DeleteDBClusterSnapshotInput{
			DBClusterSnapshotIdentifier: snapshot,
		}

		_, err := svc.DeleteDBClusterSnapshot(input)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s: %s", *snapshot, err)
		} else {
			deletedSnapshots = append(deletedSnapshots, snapshot)
			logging.Logger.Infof("Deleted RDS DB Cluster Snapshot: %s", awsgo.StringValue(snapshot))
		}
	}

	if len(deletedSnapshots) > 0 {
		for _, snapshot := range deletedSnapshots {

			err := waitUntilRdsClusterSnapshotDeleted(svc, &rds.DescribeDBClusterSnapshotsInput{
				DBClusterSnapshotIdentifier: snapshot,
			})

			if err != nil {
				logging.Logger.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}

	if len(deletedSnapshots) != len(snapshots) {
		logging.Logger.Errorf("[Failed] - %d/%d - RDS DB Cluster Snapshot(s) failed deletion in %s", len(snapshots)-len(deletedSnapshots), snapshots, *session.Config.Region)
	}

	logging.Logger.Infof("[OK] %d RDS DB Cluster Snapshot(s) deleted in %s", len(deletedSnapshots), *session.Config.Region)
	return nil
}
