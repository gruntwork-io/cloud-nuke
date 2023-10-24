package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
)

func (snapshot *RdsSnapshot) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var identifiers []*string
	err := snapshot.Client.DescribeDBSnapshotsPages(&rds.DescribeDBSnapshotsInput{}, func(page *rds.DescribeDBSnapshotsOutput, lastPage bool) bool {
		for _, s := range page.DBSnapshots {
			if configObj.RdsSnapshot.ShouldInclude(config.ResourceValue{
				Name: s.DBSnapshotIdentifier,
				Time: s.SnapshotCreateTime,
				Tags: util.ConvertRDSTagsToMap(s.TagList),
			}) {
				identifiers = append(identifiers, s.DBSnapshotIdentifier)
			}
		}

		return !lastPage
	})
	if err != nil {
		logging.Debugf("[RDS Snapshot] Failed to list snapshots: %s", err)
		return nil, err
	}

	return identifiers, nil
}

func (snapshot *RdsSnapshot) nukeAll(identifiers []*string) error {
	for _, identifier := range identifiers {
		logging.Debugf("[RDS Snapshot] Deleting %s in region %s", *identifier, snapshot.Region)
		_, err := snapshot.Client.DeleteDBSnapshot(&rds.DeleteDBSnapshotInput{
			DBSnapshotIdentifier: identifier,
		})
		if err != nil {
			logging.Errorf("[RDS Snapshot] Error deleting RDS Snapshot %s: %s", *identifier, err)
		} else {
			logging.Debugf("[RDS Snapshot] Deleted RDS Snapshot %s", *identifier)
		}
	}

	return nil
}
