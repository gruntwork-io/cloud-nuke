package resources

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// LaunchConfigs - represents all launch configurations
type LaunchConfigs struct {
	Client                   autoscalingiface.AutoScalingAPI
	Region                   string
	LaunchConfigurationNames []string
}

func (lc *LaunchConfigs) Init(session *session.Session) {
	lc.Client = autoscaling.New(session)
}

// ResourceName - the simple name of the aws resource
func (lc *LaunchConfigs) ResourceName() string {
	return "lc"
}

func (lc *LaunchConfigs) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The names of the launch configurations
func (lc *LaunchConfigs) ResourceIdentifiers() []string {
	return lc.LaunchConfigurationNames
}

func (lc *LaunchConfigs) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := lc.getAll(configObj)
	if err != nil {
		return nil, err
	}

	lc.LaunchConfigurationNames = awsgo.StringValueSlice(identifiers)
	return lc.LaunchConfigurationNames, nil
}

// Nuke - nuke 'em all!!!
func (lc *LaunchConfigs) Nuke(identifiers []string) error {
	if err := lc.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
