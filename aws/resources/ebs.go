package resources

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of EBS volume ids
func (ev *EBSVolumes) getAll(configObj config.Config) ([]*string, error) {
	// Available statuses: (creating | available | in-use | deleting | deleted | error).
	// Since the output of this function is used to delete the returned volumes
	// We want to only list EBS volumes with a status of "available" or "creating"
	// Since those are the only statuses that are eligible for deletion
	statusFilter := ec2.Filter{Name: aws.String("status"), Values: aws.StringSlice([]string{"available", "creating", "error"})}
	result, err := ev.Client.DescribeVolumes(&ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{&statusFilter},
	})

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var volumeIds []*string
	for _, volume := range result.Volumes {
		if shouldIncludeEBSVolume(volume, configObj) {
			volumeIds = append(volumeIds, volume.VolumeId)
		}
	}

	return volumeIds, nil
}

func shouldIncludeEBSVolume(volume *ec2.Volume, configObj config.Config) bool {
	name := ""
	for _, tag := range volume.Tags {
		if tag != nil && aws.StringValue(tag.Key) == "Name" {
			name = aws.StringValue(tag.Value)
		}
	}

	return configObj.EBSVolume.ShouldInclude(config.ResourceValue{
		Name: &name,
		Time: volume.CreateTime,
	})
}

// Deletes all EBS Volumes
func (ev *EBSVolumes) nukeAll(volumeIds []*string) error {
	if len(volumeIds) == 0 {
		logging.Logger.Debugf("No EBS volumes to nuke in region %s", ev.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all EBS volumes in region %s", ev.Region)
	var deletedVolumeIDs []*string

	for _, volumeID := range volumeIds {
		params := &ec2.DeleteVolumeInput{
			VolumeId: volumeID,
		}

		_, err := ev.Client.DeleteVolume(params)

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
					"region": ev.Region,
					"reason": "VolumeInUse",
				})
				logging.Logger.Debugf("EBS volume %s can't be deleted, it is still attached to an active resource", *volumeID)
			} else if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == "InvalidVolume.NotFound" {
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: "Error Nuking EBS Volume",
				}, map[string]interface{}{
					"region": ev.Region,
					"reason": "InvalidVolume.NotFound",
				})
				logging.Logger.Debugf("EBS volume %s has already been deleted", *volumeID)
			} else {
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: "Error Nuking EBS Volume",
				}, map[string]interface{}{
					"region": ev.Region,
				})
				logging.Logger.Debugf("[Failed] %s", err)
			}
		} else {
			deletedVolumeIDs = append(deletedVolumeIDs, volumeID)
			logging.Logger.Debugf("Deleted EBS Volume: %s", *volumeID)
		}
	}

	if len(deletedVolumeIDs) > 0 {
		err := ev.Client.WaitUntilVolumeDeleted(&ec2.DescribeVolumesInput{
			VolumeIds: deletedVolumeIDs,
		})
		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking EBS Volume",
			}, map[string]interface{}{
				"region": ev.Region,
			})
			return errors.WithStackTrace(err)
		}
	}

	logging.Logger.Debugf("[OK] %d EBS volumes(s) terminated in %s", len(deletedVolumeIDs), ev.Region)
	return nil
}
