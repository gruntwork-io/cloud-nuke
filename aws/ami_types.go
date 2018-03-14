package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// AMIs - represents all user owned AMIs
type AMIs struct {
	ImageIds []string
}

// ResourceName - the simple name of the aws resource
func (image AMIs) ResourceName() string {
	return "ami"
}

// ResourceIdentifiers - The AMI image ids
func (image AMIs) ResourceIdentifiers() []string {
	return image.ImageIds
}

// Nuke - nuke 'em all!!!
func (image AMIs) Nuke(session *session.Session) error {
	if err := nukeAllAMIs(session, awsgo.StringSlice(image.ImageIds)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
