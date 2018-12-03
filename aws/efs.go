package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Returns a formatted string of EFS filesystem ids
func getAllEfsFileSystems(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := efs.New(session)

	result, err := svc.DescribeFileSystems(&efs.DescribeFileSystemsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var fileSystemIds []*string
	for _, fileSystem := range result.FileSystems {
		if excludeAfter.After(*fileSystem.CreationTime) {
			fileSystemIds = append(fileSystemIds, fileSystem.FileSystemId)
		}
	}

	return fileSystemIds, nil
}

func getAllEfsMountTargets(session *session.Session, fileSystemId *string) ([]*string, error) {
	svc := efs.New(session)

	params := &efs.DescribeMountTargetsInput{
		FileSystemId: fileSystemId,
	}

	result, err := svc.DescribeMountTargets(params)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var mountTargetIds []*string
	for _, mountTarget := range result.MountTargets {
		mountTargetIds = append(mountTargetIds, mountTarget.MountTargetId)
	}

	return mountTargetIds, nil
}

// Deletes all EFS file systems
func nukeAllEfsFileSystems(session *session.Session, fileSystemIds []*string) error {
	svc := efs.New(session)

	if len(fileSystemIds) == 0 {
		logging.Logger.Infof("No EFS file systems to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all EFS file systems in region %s", *session.Config.Region)
	var deletedFileSystemIDs []*string

	for _, fileSystemID := range fileSystemIds {
		// first delete all of the associated mount targets
		mountTargetIds, err := getAllEfsMountTargets(session, fileSystemID)
		nukeAllEfsMountTargets(session, mountTargetIds)

		// TODO - we must wait for the mount targets to be deleted
		params := &efs.DeleteFileSystemInput{
			FileSystemId: fileSystemID,
		}

		_, err = svc.DeleteFileSystem(params)
		if err != nil {
			if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == "FileSystemInUse" {
				logging.Logger.Warnf("EFS file system %s cannot be deleted, it still has active mount targets", *fileSystemID)
			} else if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == "FileSystemNotFound" {
				logging.Logger.Infof("EFS file system %s has already been deleted", *fileSystemID)
			} else {
				logging.Logger.Errorf("[Failed] %s", err)
			}
		} else {
			deletedFileSystemIDs = append(deletedFileSystemIDs, fileSystemID)
			logging.Logger.Infof("Deleted EFS File System: %s", *fileSystemID)
		}
	}

	logging.Logger.Infof("[OK] %d EFS File System(s) deleted in %s", len(deletedFileSystemIDs), *session.Config.Region)
	return nil
}

// Deletes all EFS mount targets
func nukeAllEfsMountTargets(session *session.Session, mountTargetIds []*string) error {
	svc := efs.New(session)

	if len(mountTargetIds) == 0 {
		logging.Logger.Infof("No EFS mount targets to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all EFS mount targets in region %s", *session.Config.Region)
	var deletedMountTargetIDs []*string

	for _, mountTargetID := range mountTargetIds {
		params := &efs.DeleteMountTargetInput{
			MountTargetId: mountTargetID,
		}

		_, err := svc.DeleteMountTarget(params)
		if err != nil {
			if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == "MountTargetNotFound" {
				logging.Logger.Infof("EFS mount target %s has already been deleted", *mountTargetID)
			} else {
				logging.Logger.Errorf("[Failed] %s", err)
			}
		} else {
			deletedMountTargetIDs = append(deletedMountTargetIDs, mountTargetID)
			logging.Logger.Infof("Deleted EFS Mount Target: %s", *mountTargetID)
		}
	}

	logging.Logger.Infof("[OK] %d EFS Mount Target(s) deleted in %s", len(deletedMountTargetIDs), *session.Config.Region)
	return nil
}
