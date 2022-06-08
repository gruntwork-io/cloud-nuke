package aws

import (
	"fmt"
	"time"
	"github.com/gruntwork-io/go-commons/errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/gruntwork-io/go-commons/retry"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
)

func getAllEfsVolumes(session *session.Session, region string, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := efs.New(session)

	result, err := svc.DescribeFileSystems(&efs.DescribeFileSystemsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var efsVolumeIds []*string
	for _, volume := range result.FileSystems {
		if shouldIncludeEFSVolume(volume, excludeAfter, configObj) {
			efsVolumeIds = append(efsVolumeIds, volume.FileSystemId)
		}
	}

	return efsVolumeIds, nil
}

func shouldIncludeEFSVolume(volume *efs.FileSystemDescription, excludeAfter time.Time, configObj config.Config) bool {
	if volume == nil {
		return false
	}

	if excludeAfter.Before(aws.TimeValue(volume.CreationTime)) {
		return false
	}

	name := aws.StringValue(volume.Name)

	return config.ShouldInclude(
		name, 
		configObj.EFSInstances.IncludeRule.NamesRegExp,
		configObj.EFSInstances.ExcludeRule.NamesRegExp,
	)
}

// Waits until EFS volume has been successfully nuked
func waitUntilEfsVolumeIsNuked(session *session.Session, efsVolumeId *string) error {
	svc := efs.New(session)

	if efsVolumeId == nil {
		return fmt.Errorf("EFS volume ID is invalid")
	}

	// We need to delete all volume mount targets of this EFS volume before destroying the EFS volume itself.
	err := nukeEfsVolumeMountTargets(session, efsVolumeId)
	if err != nil {
		logging.Logger.Errorf("[Failed] %s", err)
	}

	_, err = svc.DeleteFileSystem(&efs.DeleteFileSystemInput{
		FileSystemId: efsVolumeId,
	})
	if err != nil {
		if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == "FileSystemNotFound" {
			logging.Logger.Infof("EFS volume %s has already been deleted", *efsVolumeId)
		} else {
			logging.Logger.Errorf("[Failed] %s", err)
		}
	}

	if err := retry.DoWithRetry(
		logging.Logger,
		fmt.Sprintf("Waiting until EFS volume ID %s is fully deleted", *efsVolumeId),
		10,
		1*time.Second,
		func () error {
			details, err := svc.DescribeFileSystems(&efs.DescribeFileSystemsInput{
				FileSystemId: efsVolumeId,
			})
			if err != nil {
				if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == "FileSystemNotFound" {
					return nil
				} else {
					return err
				}
			}
			if len(details.FileSystems) >= 1 {
				return fmt.Errorf("EFS still exists")
			}
			return nil
		},
	); err != nil {
		return err
	}

	return nil
}

func nukeEfsVolumeMountTargets(session *session.Session, efsVolumeId *string) error {
	svc := efs.New(session)
	
	mounts, err := svc.DescribeMountTargets(&efs.DescribeMountTargetsInput{
		FileSystemId: efsVolumeId,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	var deletedMountTargets []*string

	for _, mount := range mounts.MountTargets {
		_, err = svc.DeleteMountTarget(&efs.DeleteMountTargetInput{
			MountTargetId: mount.MountTargetId,
		})

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			logging.Logger.Infof("Deleted EFS Volume Mount Target: %s (volume %s)", *mount.MountTargetId, *efsVolumeId)
			deletedMountTargets = append(deletedMountTargets, mount.MountTargetId)
		}
	}

	return nil
}

// Deletes all EFS volumes
func nukeAllEfsVolumes(session *session.Session, efsVolumeIds []*string) error {
	if len(efsVolumeIds) == 0 {
		logging.Logger.Infof("No EFS volumes to nuke in this region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all EFS volumes in region %s", *session.Config.Region)
	var deletedEfsVolumes []*string

	for _, efsVolumeId := range efsVolumeIds {
		err := waitUntilEfsVolumeIsNuked(session, efsVolumeId)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			deletedEfsVolumes = append(deletedEfsVolumes, efsVolumeId)
			logging.Logger.Infof("Deleted EFS Volume: %s", *efsVolumeId)
		}
	}

	logging.Logger.Infof("[OK] %d EFS volume(s) terminated in %s", len(deletedEfsVolumes), *session.Config.Region)
	return nil
}
