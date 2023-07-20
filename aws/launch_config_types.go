package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// LaunchConfiguration - represents all launch configurations
type LaunchConfiguration struct {
	Client                   autoscalingiface.AutoScalingAPI
	Region                   string
	LaunchConfigurationNames []string
}

// ResourceName - the simple name of the aws resource
func (config LaunchConfiguration) ResourceName() string {
	return "lc"
}

func (config LaunchConfiguration) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The names of the launch configurations
func (config LaunchConfiguration) ResourceIdentifiers() []string {
	return config.LaunchConfigurationNames
}

// Nuke - nuke 'em all!!!
func (config LaunchConfiguration) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllLaunchConfigurations(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
