package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
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
	return nil
}
