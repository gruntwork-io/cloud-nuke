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

// Get all DB Instance snapshots
func getAllRdsSnapshots(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := rds.New(session)

	result, err := svc.DescribeDBSnapshots(&rds.DescribeDBSnapshotsInput{})

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var snapshots []*string

	for _, database := range result.DBSnapshots {

		// List all DB Instance Snapshot tags
		tagsResult, err := svc.ListTagsForResource(&rds.ListTagsForResourceInput{
			ResourceName: database.DBSnapshotArn,
		})

		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		if database.SnapshotCreateTime != nil && excludeAfter.After(awsgo.TimeValue(database.SnapshotCreateTime)) {
			if shouldIncludeSnapshotByName(*database.DBSnapshotIdentifier, configObj.RDSSnapshots.IncludeRule.NamesRE, configObj.RDSSnapshots.ExcludeRule.NamesRE) {
				if len(tagsResult.TagList) > 0 {
					for _, tag := range tagsResult.TagList {
						if shouldIncludeSnapshotByTag(*tag.Key, configObj.RDSSnapshots.IncludeRule.TagNamesRE, configObj.RDSSnapshots.ExcludeRule.TagNamesRE) {
							snapshots = append(snapshots, database.DBSnapshotIdentifier)
						}
					}
				} else {
					snapshots = append(snapshots, database.DBSnapshotIdentifier)
				}
			}
		}
	}

	return snapshots, nil
}

// Filter DB Instance snapshot by names_regex in config file
func shouldIncludeSnapshotByName(snapshotName string, includeNamesREList []*regexp.Regexp, excludeNamesREList []*regexp.Regexp) bool {
	shouldInclude := false

	if len(includeNamesREList) > 0 {
		// If any include rules are defined
		// and the include rule matches the snapshot, check to see if an exclude rule matches
		if includeSnapshotByNamesREList(snapshotName, includeNamesREList) {
			shouldInclude = excludeSnapshotByNamesREList(snapshotName, excludeNamesREList)
		}
	} else if len(excludeNamesREList) > 0 {
		// If there are no include rules defined, check to see if an exclude rule matches
		shouldInclude = excludeSnapshotByNamesREList(snapshotName, excludeNamesREList)
	} else {
		// Otherwise
		shouldInclude = true
	}

	return shouldInclude
}

// Include filtered DB Instance snapshot
func includeSnapshotByNamesREList(snapshotName string, reList []*regexp.Regexp) bool {
	for _, re := range reList {
		if re.MatchString(snapshotName) {
			return true
		}
	}
	return false
}

// Exclude filtered DB Instance snapshot
func excludeSnapshotByNamesREList(snapshotName string, reList []*regexp.Regexp) bool {
	for _, re := range reList {
		if re.MatchString(snapshotName) {
			return false
		}
	}
	return true
}

// Filter DB Instance snapshot by tags_regex in config file
func shouldIncludeSnapshotByTag(tagName string, includeTagNamesREList []*regexp.Regexp, excludeTagNamesREList []*regexp.Regexp) bool {
	shouldInclude := false

	if len(includeTagNamesREList) > 0 {
		// If any include rules are defined
		// and the include rule matches the snapshot tag, check to see if an exclude rule matches
		if includeSnapshotByTagsREList(tagName, includeTagNamesREList) {
			shouldInclude = excludeSnapshotByTagsREList(tagName, excludeTagNamesREList)
		}
	} else if len(excludeTagNamesREList) > 0 {
		// If there are no include rules defined, check to see if an exclude rule matches
		shouldInclude = excludeSnapshotByTagsREList(tagName, excludeTagNamesREList)
	} else {
		// Otherwise
		shouldInclude = true
	}

	return shouldInclude
}

// Include filtered DB Instance
func includeSnapshotByTagsREList(tagName string, reList []*regexp.Regexp) bool {
	for _, re := range reList {
		if re.MatchString(tagName) {
			return true
		}
	}
	return false
}

// Exclude filtered DB Instance
func excludeSnapshotByTagsREList(tagName string, reList []*regexp.Regexp) bool {
	for _, re := range reList {
		if re.MatchString(tagName) {
			return false
		}
	}
	return true
}

// Nuke-Delete all DB Instance snapshots
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
		logging.Logger.Errorf("[Failed] - %d/%d - RDS DB Snapshot(s) failed deletion in %s", len(snapshots)-len(deletedSnapshots), len(snapshots), *session.Config.Region)
	}

	logging.Logger.Infof("[OK] %d RDS DB Snapshot(s) deleted in %s", len(deletedSnapshots), *session.Config.Region)
	return nil
}
