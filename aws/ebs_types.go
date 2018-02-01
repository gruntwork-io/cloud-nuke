package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// EBSVolumes - represents all ebs volumes
type EBSVolumes struct {
	VolumeIds []string
}

// ResourceName - the simple name of the aws resource
func (volume EBSVolumes) ResourceName() string {
	return "ebs"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (volume EBSVolumes) ResourceIdentifiers() []string {
	return volume.VolumeIds
}

// Nuke - nuke 'em all!!!
func (volume EBSVolumes) Nuke(session *session.Session) error {
	if err := nukeAllEbsVolumes(session, awsgo.StringSlice(volume.VolumeIds)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
