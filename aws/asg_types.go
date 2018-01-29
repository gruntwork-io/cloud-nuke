package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// ASGroups - represents all ec2 instances
type ASGroups struct {
	GroupNames []string
}

// ResourceName - the simple name of the aws resource
func (group ASGroups) ResourceName() string {
	return "asg"
}

// ResourceIdentifiers - The group names of the auto scaling groups
func (group ASGroups) ResourceIdentifiers() []string {
	return group.GroupNames
}

// Nuke - nuke 'em all!!!
func (group ASGroups) Nuke(session *session.Session) error {
	if err := nukeAllAutoScalingGroups(session, awsgo.StringSlice(group.GroupNames)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
