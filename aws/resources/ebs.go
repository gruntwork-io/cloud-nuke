package resources

import (
	"context"
	goerr "errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of EBS volume ids
func (ev *EBSVolumes) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	// Available statuses: (creating | available | in-use | deleting | deleted | error).
	// Since the output of this function is used to delete the returned volumes
	// We want to only list EBS volumes with a status of "available" or "creating"
	// Since those are the only statuses that are eligible for deletion
	statusFilter := types.Filter{Name: aws.String("status"), Values: []string{"available", "creating", "error"}}
	result, err := ev.Client.DescribeVolumes(ev.Context, &ec2.DescribeVolumesInput{
		Filters: []types.Filter{statusFilter},
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

	// checking the nukable permissions
	ev.VerifyNukablePermissions(volumeIds, func(id *string) error {
		_, err := ev.Client.DeleteVolume(ev.Context, &ec2.DeleteVolumeInput{
			VolumeId: id,
			DryRun:   aws.Bool(true),
		})
		return err
	})

	return volumeIds, nil
}

func shouldIncludeEBSVolume(volume types.Volume, configObj config.Config) bool {
	name := ""
	for _, tag := range volume.Tags {
		if aws.ToString(tag.Key) == "Name" {
			name = aws.ToString(tag.Value)
		}
	}

	return configObj.EBSVolume.ShouldInclude(config.ResourceValue{
		Name: &name,
		Time: volume.CreateTime,
		Tags: util.ConvertTypesTagsToMap(volume.Tags),
	})
}

// Deletes all EBS Volumes
func (ev *EBSVolumes) nukeAll(volumeIds []*string) error {
	if len(volumeIds) == 0 {
		logging.Debugf("No EBS volumes to nuke in region %s", ev.Region)
		return nil
	}

	logging.Debugf("Deleting all EBS volumes in region %s", ev.Region)
	var deletedVolumeIDs []*string

	for _, volumeID := range volumeIds {

		if nukable, reason := ev.IsNukable(aws.ToString(volumeID)); !nukable {
			logging.Debugf("[Skipping] %s nuke because %v", aws.ToString(volumeID), reason)
			continue
		}

		params := &ec2.DeleteVolumeInput{
			VolumeId: volumeID,
		}

		_, err := ev.Client.DeleteVolume(ev.Context, params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(volumeID),
			ResourceType: "EBS Volume",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			var apiErr smithy.APIError
			if goerr.As(err, &apiErr) {
				switch apiErr.ErrorCode() {
				case "VolumeInUse":
					logging.Debugf("EBS volume %s can't be deleted, it is still attached to an active resource", *volumeID)
				case "InvalidVolume.NotFound":
					logging.Debugf("EBS volume %s has already been deleted", *volumeID)
				default:
					logging.Debugf("[Failed] %s", err)
				}
			}
		} else {
			deletedVolumeIDs = append(deletedVolumeIDs, volumeID)
			logging.Debugf("Deleted EBS Volume: %s", *volumeID)
		}
	}

	if len(deletedVolumeIDs) > 0 {
		waiter := ec2.NewVolumeDeletedWaiter(ev.Client)
		err := waiter.Wait(ev.Context, &ec2.DescribeVolumesInput{
			VolumeIds: aws.ToStringSlice(deletedVolumeIDs),
		}, ev.Timeout)
		if err != nil {
			logging.Debugf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
	}

	logging.Debugf("[OK] %d EBS volumes(s) terminated in %s", len(deletedVolumeIDs), ev.Region)
	return nil
}
