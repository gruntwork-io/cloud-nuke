package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type LaunchConfigsAPI interface {
	DeleteLaunchConfiguration(ctx context.Context, params *autoscaling.DeleteLaunchConfigurationInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DeleteLaunchConfigurationOutput, error)
	DescribeLaunchConfigurations(ctx context.Context, params *autoscaling.DescribeLaunchConfigurationsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeLaunchConfigurationsOutput, error)
}

// LaunchConfigs - represents all launch configurations
type LaunchConfigs struct {
	BaseAwsResource
	Client                   LaunchConfigsAPI
	Region                   string
	LaunchConfigurationNames []string
}

func (lc *LaunchConfigs) InitV2(cfg aws.Config) {
	lc.Client = autoscaling.NewFromConfig(cfg)
}

func (lc *LaunchConfigs) IsUsingV2() bool { return true }

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

func (lc *LaunchConfigs) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.LaunchConfiguration
}

func (lc *LaunchConfigs) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := lc.getAll(configObj)
	if err != nil {
		return nil, err
	}

	lc.LaunchConfigurationNames = aws.ToStringSlice(identifiers)
	return lc.LaunchConfigurationNames, nil
}

// Nuke - nuke 'em all!!!
func (lc *LaunchConfigs) Nuke(identifiers []string) error {
	if err := lc.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
