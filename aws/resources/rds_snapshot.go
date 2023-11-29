package resources

import (
	"context"

	"github.com/andrewderr/cloud-nuke-a1/config"
	"github.com/andrewderr/cloud-nuke-a1/logging"
	"github.com/andrewderr/cloud-nuke-a1/report"
	"github.com/andrewderr/cloud-nuke-a1/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
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

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(identifier),
			ResourceType: snapshot.ResourceName(),
			Error:        err,
		}
		report.Record(e)
	}

	return nil
}
