package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/go-commons/errors"
)

// AMI - represents all user owned AMI
type AMI struct {
	Client   ec2iface.EC2API
	Region   string
	ImageIds []string
}

// ResourceName - the simple name of the aws resource
func (image AMI) ResourceName() string {
	return "ami"
}

// ResourceIdentifiers - The AMI image ids
func (image AMI) ResourceIdentifiers() []string {
	return image.ImageIds
}

func (image AMI) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (image AMI) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllAMIs(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type ImageAvailableError struct{}

func (e ImageAvailableError) Error() string {
	return "Image didn't become available within wait attempts"
}
