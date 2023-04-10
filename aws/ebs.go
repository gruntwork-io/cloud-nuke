package aws

import (
	"time"

	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of EBS volume ids
func getAllEbsVolumes(session *session.Session, region string, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := ec2.New(session)

	// Available statuses: (creating | available | in-use | deleting | deleted | error).
	// Since the output of this function is used to delete the returned volumes
	// We want to only list EBS volumes with a status of "available" or "creating"
	// Since those are the only statuses that are eligible for deletion
	status_filter := ec2.Filter{Name: aws.String("status"), Values: []*string{aws.String("available"), aws.String("creating"), aws.String("error")}}

	result, err := svc.DescribeVolumes(&ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{&status_filter},
	})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var volumeIds []*string
	for _, volume := range result.Volumes {
		if shouldIncludeEBSVolume(volume, excludeAfter, configObj) {
			volumeIds = append(volumeIds, volume.VolumeId)
		}
	}

	return volumeIds, nil
}

func shouldIncludeEBSVolume(volume *ec2.Volume, excludeAfter time.Time, configObj config.Config) bool {
	if volume == nil {
		return false
	}

	if excludeAfter.Before(aws.TimeValue(volume.CreateTime)) {
		return false
	}

	name := ""
	for _, tag := range volume.Tags {
		if tag != nil && aws.StringValue(tag.Key) == "Name" {
			name = aws.StringValue(tag.Value)
		}
	}
	return config.ShouldInclude(
		name,
		configObj.EBSVolume.IncludeRule.NamesRegExp,
		configObj.EBSVolume.ExcludeRule.NamesRegExp,
	)
}

// Deletes all EBS Volumes
func nukeAllEbsVolumes(session *session.Session, volumeIds []*string) error {
	svc := ec2.New(session)

	if len(volumeIds) == 0 {
		logging.Logger.Debugf("No EBS volumes to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all EBS volumes in region %s", *session.Config.Region)
	var deletedVolumeIDs []*string

	for _, volumeID := range volumeIds {
		params := &ec2.DeleteVolumeInput{
			VolumeId: volumeID,
		}

		_, err := svc.DeleteVolume(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(volumeID),
			ResourceType: "EBS Volume",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == "VolumeInUse" {
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: "Error Nuking EBS Volume",
				}, map[string]interface{}{
					"region": *session.Config.Region,
					"reason": "VolumeInUse",
				})
				logging.Logger.Debugf("EBS volume %s can't be deleted, it is still attached to an active resource", *volumeID)
			} else if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == "InvalidVolume.NotFound" {
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: "Error Nuking EBS Volume",
				}, map[string]interface{}{
					"region": *session.Config.Region,
					"reason": "InvalidVolume.NotFound",
				})
				logging.Logger.Debugf("EBS volume %s has already been deleted", *volumeID)
			} else {
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: "Error Nuking EBS Volume",
				}, map[string]interface{}{
					"region": *session.Config.Region,
				})
				logging.Logger.Debugf("[Failed] %s", err)
			}
		} else {
			deletedVolumeIDs = append(deletedVolumeIDs, volumeID)
			logging.Logger.Debugf("Deleted EBS Volume: %s", *volumeID)
		}
	}

	if len(deletedVolumeIDs) > 0 {
		err := svc.WaitUntilVolumeDeleted(&ec2.DescribeVolumesInput{
			VolumeIds: deletedVolumeIDs,
		})
		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking EBS Volume",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
			return errors.WithStackTrace(err)
		}
	}

	logging.Logger.Debugf("[OK] %d EBS volumes(s) terminated in %s", len(deletedVolumeIDs), *session.Config.Region)
	return nil
}
