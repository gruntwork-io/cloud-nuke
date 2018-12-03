package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// EFSFileSystems - represents all EFS file systems
type EFSFileSystems struct {
	FileSystemIds []string
}

// ResourceName - the simple name of the aws resource
func (fileSystem EFSFileSystems) ResourceName() string {
	return "efs"
}

// ResourceIdentifiers - the file system ids of the EFS file systems
func (fileSystem EFSFileSystems) ResourceIdentifiers() []string {
	return fileSystem.FileSystemIds
}

// MaxBatchSize - the tentative batch size to ensure AWS doesn't throttle
func (fileSystem EFSFileSystems) MaxBatchSize() int {
	return 200
}

// Nuke - nuke 'em all!!!
func (fileSystem EFSFileSystems) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllEfsFileSystems(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// EFSMountTargets - represents all EFS mount targets
type EFSMountTargets struct {
	MountTargetIds []string
}

// ResourceName - the simple name of the aws resource
func (mountTarget EFSMountTargets) ResourceName() string {
	return "efsmt"
}

// ResourceIdentifiers - the file system ids of the EFS file systems
func (mountTarget EFSMountTargets) ResourceIdentifiers() []string {
	return mountTarget.MountTargetIds
}

// MaxBatchSize - the tentative batch size to ensure AWS doesn't throttle
func (mountTarget EFSMountTargets) MaxBatchSize() int {
	return 200
}

// Nuke - nuke 'em all!!!
func (mountTarget EFSMountTargets) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllEfsMountTargets(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
