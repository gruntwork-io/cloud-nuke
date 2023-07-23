package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/go-commons/errors"
)

// LaunchTemplates - represents all launch templates
type LaunchTemplates struct {
	Client              ec2iface.EC2API
	Region              string
	LaunchTemplateNames []string
}

// ResourceName - the simple name of the aws resource
func (template LaunchTemplates) ResourceName() string {
	return "lt"
}

func (template LaunchTemplates) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The names of the launch templates
func (template LaunchTemplates) ResourceIdentifiers() []string {
	return template.LaunchTemplateNames
}

// Nuke - nuke 'em all!!!
func (template LaunchTemplates) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllLaunchTemplates(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
