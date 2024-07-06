package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of Snapshot snapshot ids
func (s *Snapshots) getAll(c context.Context, configObj config.Config) ([]*string, error) {

	// status - The status of the s (pending | completed | error).
	// Since the output of this function is used to delete the returned snapshots
	// We only want to list EBS Snapshots with a status of "completed"
	// Since that is the only status that is eligible for deletion
	status_filter := ec2.Filter{Name: awsgo.String("status"), Values: aws.StringSlice([]string{"completed", "error"})}
	params := &ec2.DescribeSnapshotsInput{
		OwnerIds: []*string{awsgo.String("self")},
		Filters:  []*ec2.Filter{&status_filter},
	}

	output, err := s.Client.DescribeSnapshotsWithContext(s.Context, params)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var snapshotIds []*string
	for _, snapshot := range output.Snapshots {
		if configObj.Snapshots.ShouldInclude(config.ResourceValue{
			Time: snapshot.StartTime,
			Tags: util.ConvertEC2TagsToMap(snapshot.Tags),
		}) && !SnapshotHasAWSBackupTag(snapshot.Tags) {
			snapshotIds = append(snapshotIds, snapshot.SnapshotId)
		}
	}

	// checking the nukable permissions
	s.VerifyNukablePermissions(snapshotIds, func(id *string) error {
		_, err := s.Client.DeleteSnapshotWithContext(s.Context, &ec2.DeleteSnapshotInput{
			SnapshotId: id,
			DryRun:     awsgo.Bool(true),
		})
		return err
	})

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

func (s *Snapshots) nuke(id *string) error {

	if err := s.nukeAMI(id); err != nil {
		return errors.WithStackTrace(err)
	}

	if err := s.nukeSnapshot(id); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (s *Snapshots) nukeSnapshot(snapshotID *string) error {
	_, err := s.Client.DeleteSnapshotWithContext(s.Context, &ec2.DeleteSnapshotInput{
		SnapshotId: snapshotID,
	})
	return err
}

// nuke AMI which created from the snapshot
func (s *Snapshots) nukeAMI(snapshotID *string) error {
	logging.Debugf("De-registering images for snapshot: %s", awsgo.StringValue(snapshotID))

	output, err := s.Client.DescribeImagesWithContext(s.Context, &ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			{
				Name:   awsgo.String("block-device-mapping.snapshot-id"),
				Values: []*string{snapshotID},
			},
		},
	})

	if err != nil {
		logging.Debugf("[Describe Images] Failed to describe images for snapshot: %s", awsgo.StringValue(snapshotID))
		return err
	}

	for _, image := range output.Images {
		_, err := s.Client.DeregisterImageWithContext(s.Context, &ec2.DeregisterImageInput{
			ImageId: image.ImageId,
		})

		if err != nil {
			logging.Debugf("[Failed] de-registering image %v for snapshot: %s", awsgo.StringValue(image.ImageId), awsgo.StringValue(snapshotID))
			return err
		}
	}

	logging.Debugf("[Ok] De-registered all the images for snapshot: %s", awsgo.StringValue(snapshotID))

	return nil
}

// Deletes all Snapshots
func (s *Snapshots) nukeAll(snapshotIds []*string) error {

	if len(snapshotIds) == 0 {
		logging.Debugf("No Snapshots to nuke in region %s", s.Region)
		return nil
	}

	logging.Debugf("Deleting all Snapshots in region %s", s.Region)
	var deletedSnapshotIDs []*string

	for _, snapshotID := range snapshotIds {

		if nukable, reason := s.IsNukable(awsgo.StringValue(snapshotID)); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", awsgo.StringValue(snapshotID), reason)
			continue
		}

		err := s.nuke(snapshotID)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(snapshotID),
			ResourceType: "EBS Snapshot",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedSnapshotIDs = append(deletedSnapshotIDs, snapshotID)
			logging.Debugf("Deleted Snapshot: %s", *snapshotID)
		}
	}

	logging.Debugf("[OK] %d Snapshot(s) terminated in %s", len(deletedSnapshotIDs), s.Region)
	return nil
}
