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

func (image AMIs) MaxBatchSize() int {
	return 200
}

// Nuke - nuke 'em all!!!
func (image AMIs) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllAMIs(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type ImageAvailableError struct{}

func (e ImageAvailableError) Error() string {
	return "Image didn't become available within wait attempts"
}
