package aws

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/aws-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Returns a formatted string of EBS volume ids
func getAllEbsVolumes(session *session.Session, region string) ([]*string, error) {
	svc := ec2.New(session)

	result, err := svc.DescribeVolumes(&ec2.DescribeVolumesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var volumeIds []*string
	for _, volume := range result.Volumes {
		volumeIds = append(volumeIds, volume.VolumeId)
	}

	return volumeIds, nil
}

// Deletes all EBS Volumes
func nukeAllEbsVolumes(session *session.Session, volumeIds []*string) error {
	svc := ec2.New(session)

	if len(volumeIds) == 0 {
		logging.Logger.Infof("No EBS volumes to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all EBS volumes in region %s", *session.Config.Region)

	for _, volumeID := range volumeIds {
		params := &ec2.DeleteVolumeInput{
			VolumeId: volumeID,
		}

		_, err := svc.DeleteVolume(params)
		if err != nil {
			// Ignore not found errors, some volumes are deleted along with EC2 Instances
			if !strings.Contains(err.Error(), "InvalidVolume.NotFound") {
				logging.Logger.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}

			logging.Logger.Infof("EBS volume %s has already been deleted", *volumeID)
		} else {
			logging.Logger.Infof("Deleted EBS Volume: %s", *volumeID)
		}
	}

	err := svc.WaitUntilVolumeDeleted(&ec2.DescribeVolumesInput{
		VolumeIds: volumeIds,
	})

	if err != nil {
		return errors.WithStackTrace(err)
	}

	logging.Logger.Infof("[OK] %d EBS volumes(s) terminated in %s", len(volumeIds), *session.Config.Region)
	return nil
}
