package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// NewLaunchConfigs creates a new Launch Configurations resource using the generic resource pattern.
func NewLaunchConfigs() AwsResource {
	return NewAwsResource(&resource.Resource[*autoscaling.Client]{
		ResourceTypeName: "lc",
		BatchSize:        49,
		InitClient: func(r *resource.Resource[*autoscaling.Client], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for AutoScaling client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = autoscaling.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.LaunchConfiguration
		},
		Lister: listLaunchConfigs,
		Nuker:  resource.SimpleBatchDeleter(deleteLaunchConfig),
	})
}

// listLaunchConfigs retrieves all launch configurations that match the config filters.
func listLaunchConfigs(ctx context.Context, client *autoscaling.Client, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	result, err := client.DescribeLaunchConfigurations(ctx, &autoscaling.DescribeLaunchConfigurationsInput{})
	if err != nil {
		return nil, err
	}

	var names []*string
	for _, lc := range result.LaunchConfigurations {
		if cfg.ShouldInclude(config.ResourceValue{
			Time: lc.CreatedTime,
			Name: lc.LaunchConfigurationName,
		}) {
			names = append(names, lc.LaunchConfigurationName)
		}
	}

	return names, nil
}

// deleteLaunchConfig deletes a single launch configuration.
func deleteLaunchConfig(ctx context.Context, client *autoscaling.Client, name *string) error {
	_, err := client.DeleteLaunchConfiguration(ctx, &autoscaling.DeleteLaunchConfigurationInput{
		LaunchConfigurationName: name,
	})
	if err != nil {
		return err
	}

	logging.Debugf("Deleted Launch Configuration: %s", aws.ToString(name))
	return nil
}
