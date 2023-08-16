package resources

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// ASGroups - represents all auto scaling groups
type ASGroups struct {
	Client     autoscalingiface.AutoScalingAPI
	Region     string
	GroupNames []string
}

func (ag *ASGroups) Init(session *session.Session) {
	ag.Client = autoscaling.New(session)
}

// ResourceName - the simple name of the aws resource
func (ag *ASGroups) ResourceName() string {
	return "asg"
}

func (ag *ASGroups) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The group names of the auto scaling groups
func (ag *ASGroups) ResourceIdentifiers() []string {
	return ag.GroupNames
}

func (ag *ASGroups) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := ag.getAll(configObj)
	if err != nil {
		return nil, err
	}

	ag.GroupNames = awsgo.StringValueSlice(identifiers)
	return ag.GroupNames, nil
}

// Nuke - nuke 'em all!!!
func (ag *ASGroups) Nuke(identifiers []string) error {
	if err := ag.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
