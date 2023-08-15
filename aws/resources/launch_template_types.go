package resources

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// LaunchTemplates - represents all launch templates
type LaunchTemplates struct {
	Client              ec2iface.EC2API
	Region              string
	LaunchTemplateNames []string
}

func (lt *LaunchTemplates) Init(session *session.Session) {
	lt.Client = ec2.New(session)
}

// ResourceName - the simple name of the aws resource
func (lt *LaunchTemplates) ResourceName() string {
	return "lt"
}

func (lt *LaunchTemplates) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The names of the launch templates
func (lt *LaunchTemplates) ResourceIdentifiers() []string {
	return lt.LaunchTemplateNames
}

func (lt *LaunchTemplates) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := lt.getAll(configObj)
	if err != nil {
		return nil, err
	}

	lt.LaunchTemplateNames = awsgo.StringValueSlice(identifiers)
	return lt.LaunchTemplateNames, nil
}

// Nuke - nuke 'em all!!!
func (lt *LaunchTemplates) Nuke(identifiers []string) error {
	if err := lt.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
