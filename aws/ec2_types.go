package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// EC2Instances - represents all ec2 instances
type EC2Instances struct {
	InstanceIds []string
}

// ResourceName - the simple name of the aws resource
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

// NukeBatch - nuke some!!!
func (instance EC2Instances) NukeBatch(session *session.Session, identifiers []string) error {
	if err := nukeAllEc2Instances(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
