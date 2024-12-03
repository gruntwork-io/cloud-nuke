package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
)

func (snapshot *RdsSnapshot) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var identifiers []*string

	// Initialize the paginator
	paginator := rds.NewDescribeDBSnapshotsPaginator(snapshot.Client, &rds.DescribeDBSnapshotsInput{})

	// Iterate through the pages
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			logging.Debugf("[RDS Snapshot] Failed to list snapshots: %s", err)
			return nil, err
		}

		// Process each snapshot in the current page
		for _, s := range page.DBSnapshots {
			if configObj.RdsSnapshot.ShouldInclude(config.ResourceValue{
				Name: s.DBSnapshotIdentifier,
				Time: s.SnapshotCreateTime,
				Tags: util.ConvertRDSTypeTagsToMap(s.TagList),
			}) {
				identifiers = append(identifiers, s.DBSnapshotIdentifier)
			}
		}
	}

	return identifiers, nil
}

func (snapshot *RdsSnapshot) nukeAll(identifiers []*string) error {
	for _, identifier := range identifiers {
		logging.Debugf("[RDS Snapshot] Deleting %s in region %s", *identifier, snapshot.Region)
		_, err := snapshot.Client.DeleteDBSnapshot(snapshot.Context, &rds.DeleteDBSnapshotInput{
			DBSnapshotIdentifier: identifier,
		})
		if err != nil {
			logging.Errorf("[RDS Snapshot] Error deleting RDS Snapshot %s: %s", *identifier, err)
		} else {
			logging.Debugf("[RDS Snapshot] Deleted RDS Snapshot %s", *identifier)
		}

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(identifier),
			ResourceType: snapshot.ResourceName(),
			Error:        err,
		}
		report.Record(e)
	}

	return nil
}
