package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Name - the simple name of the aws resource
func (instance EC2Instances) ResourceName() string {
	return "ec2"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (instance EC2Instances) ResourceIdentifiers() []string {
	return instance.InstanceIds
}

// Nuke - nuke 'em all!!!
func (instance EC2Instances) Nuke(session *session.Session) error {
	if err := nukeAllEc2Instances(session, awsgo.StringSlice(instance.InstanceIds)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
