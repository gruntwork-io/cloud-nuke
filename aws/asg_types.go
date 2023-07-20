package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// AutoScalingGroup - represents all auto scaling groups
type AutoScalingGroup struct {
	Client     autoscalingiface.AutoScalingAPI
	Region     string
	GroupNames []string
}

// ResourceName - the simple name of the aws resource
func (group AutoScalingGroup) ResourceName() string {
	return "asg"
}

func (group AutoScalingGroup) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The group names of the auto scaling groups
func (group AutoScalingGroup) ResourceIdentifiers() []string {
	return group.GroupNames
}

// Nuke - nuke 'em all!!!
func (group AutoScalingGroup) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllAutoScalingGroups(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
