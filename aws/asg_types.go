package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// ASGroups - represents all auto scaling groups
type ASGroups struct {
	GroupNames []string
}

// ResourceName - the simple name of the aws resource
func (group ASGroups) ResourceName() string {
	return "asg"
}

func (group ASGroups) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The group names of the auto scaling groups
func (group ASGroups) ResourceIdentifiers() []string {
	return group.GroupNames
}

// Nuke - nuke 'em all!!!
func (group ASGroups) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllAutoScalingGroups(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
