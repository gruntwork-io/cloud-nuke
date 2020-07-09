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

// Get all DB Cluster snapshots
func getAllRdsClusterSnapshots(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := rds.New(session)

	result, err := svc.DescribeDBClusterSnapshots(&rds.DescribeDBClusterSnapshotsInput{})

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var snapshots []*string

	for _, snapshot := range result.DBClusterSnapshots {

		// List all DB Cluster Snapshot tags
		tagsResult, err := svc.ListTagsForResource(&rds.ListTagsForResourceInput{
			ResourceName: snapshot.DBClusterSnapshotArn,
		})

		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		// Automated DB snapshots can only be deleted by deleting the DB instance or
		// changing the backup retention period for the DB instance to 0.
		// This edge case can't be handled since all DB instance related automated snapshots will be deleted.
		// Refer to https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_DeleteSnapshot.html
		if *snapshot.SnapshotType != "automated" {
			if snapshot.SnapshotCreateTime != nil && excludeAfter.After(awsgo.TimeValue(snapshot.SnapshotCreateTime)) {
				if shouldIncludeClusterSnapshotByName(*snapshot.DBClusterSnapshotIdentifier, configObj.RDSSnapshots.IncludeRule.NamesRE, configObj.RDSSnapshots.ExcludeRule.NamesRE) {
					if len(tagsResult.TagList) > 0 {
						for _, tag := range tagsResult.TagList {
							if shouldIncludeClusterSnapshotByTag(*tag.Key, configObj.RDSSnapshots.IncludeRule.TagNamesRE, configObj.RDSSnapshots.ExcludeRule.TagNamesRE) {
								snapshots = append(snapshots, snapshot.DBClusterSnapshotIdentifier)
							}
						}
					} else {
						snapshots = append(snapshots, snapshot.DBClusterSnapshotIdentifier)
					}
				}
			}
		}
	}

	return snapshots, nil
}

// Filter DB Cluster snapshot by names_regex in config file
func shouldIncludeClusterSnapshotByName(snapshotName string, includeNamesREList []*regexp.Regexp, excludeNamesREList []*regexp.Regexp) bool {
	shouldInclude := false

	if len(includeNamesREList) > 0 {
		// If any include rules are defined
		// and the include rule matches the snapshot, check to see if an exclude rule matches
		if includeClusterSnapshotByNamesREList(snapshotName, includeNamesREList) {
			shouldInclude = excludeClusterSnapshotByNamesREList(snapshotName, excludeNamesREList)
		}
	} else if len(excludeNamesREList) > 0 {
		// If there are no include rules defined, check to see if an exclude rule matches
		shouldInclude = excludeClusterSnapshotByNamesREList(snapshotName, excludeNamesREList)
	} else {
		// Otherwise
		shouldInclude = true
	}

	return shouldInclude
}

// Include filtered DB Cluster snapshot if name matches
func includeClusterSnapshotByNamesREList(snapshotName string, reList []*regexp.Regexp) bool {
	for _, re := range reList {
		if re.MatchString(snapshotName) {
			return true
		}
	}
	return false
}

// Exclude filtered DB Cluster snapshot if name matches
func excludeClusterSnapshotByNamesREList(snapshotName string, reList []*regexp.Regexp) bool {
	for _, re := range reList {
		if re.MatchString(snapshotName) {
			return false
		}
	}
	return true
}

// Filter DB Cluster snapshot by tags_regex in config file
func shouldIncludeClusterSnapshotByTag(tagName string, includeTagNamesREList []*regexp.Regexp, excludeTagNamesREList []*regexp.Regexp) bool {
	shouldInclude := false

	if len(includeTagNamesREList) > 0 {
		// If any include rules are defined
		// and the include rule matches the snapshot tag, check to see if an exclude rule matches
		if includeClusterSnapshotByTagsREList(tagName, includeTagNamesREList) {
			shouldInclude = excludeClusterSnapshotByTagsREList(tagName, excludeTagNamesREList)
		}
	} else if len(excludeTagNamesREList) > 0 {
		// If there are no include rules defined, check to see if an exclude rule matches
		shouldInclude = excludeClusterSnapshotByTagsREList(tagName, excludeTagNamesREList)
	} else {
		// Otherwise
		shouldInclude = true
	}

	return shouldInclude
}

// Include filtered DB Cluster if tag matches
func includeClusterSnapshotByTagsREList(tagName string, reList []*regexp.Regexp) bool {
	for _, re := range reList {
		if re.MatchString(tagName) {
			return true
		}
	}
	return false
}

// Exclude filtered DB Cluster if tag matches
func excludeClusterSnapshotByTagsREList(tagName string, reList []*regexp.Regexp) bool {
	for _, re := range reList {
		if re.MatchString(tagName) {
			return false
		}
	}
	return true
}

// Nuke-Delete all DB Cluster snapshots
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
		logging.Logger.Errorf("[Failed] - %d/%d - RDS DB Cluster Snapshot(s) failed deletion in %s", len(snapshots)-len(deletedSnapshots), len(snapshots), *session.Config.Region)
	}

	logging.Logger.Infof("[OK] %d RDS DB Cluster Snapshot(s) deleted in %s", len(deletedSnapshots), *session.Config.Region)
	return nil
}
