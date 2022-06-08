package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/gruntwork-io/cloud-nuke/config"
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

	return efsVolumeIds
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
		configObj.EFSInstances.ExcludeRule.NamesRegExp
	)
}

func nukeAllEfsVolumes(session *session.Session, efsVolumeIds []*string) error {
	svc := efs.New(session)

	if len(efsVolumeIds) == 0 {
		logging.Logger.Infof("No EFS volumes to nuke in this region %s", *session.Config.Region)
	}

	logging.Logger.Infof("Deleting all EFS volumes in region %s", *session.Config.Region)
	var deletedEfsVolumes []*string

	for _, efsVolumeID := range efsVolumeIds {
		params := &efs.DeleteFileSystemInput{
			FileSystemId: efsVolumeID
		}

		_, err := svc.DeleteFileSystem(params)
		if err != nil {
			if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == "FileSystemNotFound" {
				logging.Logger.Infof("EFS volume %s has already been deleted", *efsVolumeID)
			} else {
				logging.Logger.Errorf("[Failed] %s", err)
			}
		} else {
			deletedEfsVolumes = append(deletedEfsVolumes, *efsVolumeID)
			logging.Logger.Infof("Deleted EFS Volume: %s", *efsVolumeID)
		}
	}

	logging.Logger.Infof("[OK] %s EFS volume(s) terminated in %s", len(deletedEfsVolumes), *session.Config.Region)
	return nil
}
