package resources

import (
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of Snapshot snapshot ids
func (s *Snapshots) getAll(configObj config.Config) ([]*string, error) {

	// status - The status of the s (pending | completed | error).
	// Since the output of this function is used to delete the returned snapshots
	// We only want to list EBS Snapshots with a status of "completed"
	// Since that is the only status that is eligible for deletion
	status_filter := ec2.Filter{Name: awsgo.String("status"), Values: aws.StringSlice([]string{"completed", "error"})}
	params := &ec2.DescribeSnapshotsInput{
		OwnerIds: []*string{awsgo.String("self")},
		Filters:  []*ec2.Filter{&status_filter},
	}

	output, err := s.Client.DescribeSnapshots(params)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var snapshotIds []*string
	for _, snapshot := range output.Snapshots {
		if configObj.Snapshots.ShouldInclude(config.ResourceValue{Time: snapshot.StartTime}) &&
			!SnapshotHasAWSBackupTag(snapshot.Tags) {
			snapshotIds = append(snapshotIds, snapshot.SnapshotId)
		}
	}

	return snapshotIds, nil
}

// Check if the image has an AWS Backup tag
// Resources created by AWS Backup are listed as owned by self, but are actually
// AWS managed resources and cannot be deleted here.
func SnapshotHasAWSBackupTag(tags []*ec2.Tag) bool {
	t := make(map[string]string)

	for _, v := range tags {
		t[awsgo.StringValue(v.Key)] = awsgo.StringValue(v.Value)
	}

	if _, ok := t["aws:backup:source-resource"]; ok {
		return true
	}
	return false
}

// Deletes all Snapshots
func (s *Snapshots) nukeAll(snapshotIds []*string) error {

	if len(snapshotIds) == 0 {
		logging.Logger.Debugf("No Snapshots to nuke in region %s", s.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all Snapshots in region %s", s.Region)
	var deletedSnapshotIDs []*string

	for _, snapshotID := range snapshotIds {
		params := &ec2.DeleteSnapshotInput{
			SnapshotId: snapshotID,
		}

		_, err := s.Client.DeleteSnapshot(params)

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
				"region": s.Region,
			})
		} else {
			deletedSnapshotIDs = append(deletedSnapshotIDs, snapshotID)
			logging.Logger.Debugf("Deleted Snapshot: %s", *snapshotID)
		}
	}

	logging.Logger.Debugf("[OK] %d Snapshot(s) terminated in %s", len(deletedSnapshotIDs), s.Region)
	return nil
}
