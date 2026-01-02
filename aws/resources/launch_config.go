package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// LaunchConfigsAPI defines the interface for Launch Configuration operations.
type LaunchConfigsAPI interface {
	DescribeLaunchConfigurations(ctx context.Context, params *autoscaling.DescribeLaunchConfigurationsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeLaunchConfigurationsOutput, error)
	DeleteLaunchConfiguration(ctx context.Context, params *autoscaling.DeleteLaunchConfigurationInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DeleteLaunchConfigurationOutput, error)
}

// NewLaunchConfigs creates a new Launch Configurations resource using the generic resource pattern.
func NewLaunchConfigs() AwsResource {
	return NewAwsResource(&resource.Resource[LaunchConfigsAPI]{
		ResourceTypeName: "lc",
		BatchSize:        49,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[LaunchConfigsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = autoscaling.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.LaunchConfiguration
		},
		Lister: listLaunchConfigs,
		Nuker:  resource.SimpleBatchDeleter(deleteLaunchConfig),
	})
}

// listLaunchConfigs retrieves all launch configurations that match the config filters.
func listLaunchConfigs(ctx context.Context, client LaunchConfigsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var names []*string

	paginator := autoscaling.NewDescribeLaunchConfigurationsPaginator(client, &autoscaling.DescribeLaunchConfigurationsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, lc := range page.LaunchConfigurations {
			if cfg.ShouldInclude(config.ResourceValue{
				Time: lc.CreatedTime,
				Name: lc.LaunchConfigurationName,
			}) {
				names = append(names, lc.LaunchConfigurationName)
			}
		}
	}

	return names, nil
}

// deleteLaunchConfig deletes a single launch configuration.
func deleteLaunchConfig(ctx context.Context, client LaunchConfigsAPI, name *string) error {
	_, err := client.DeleteLaunchConfiguration(ctx, &autoscaling.DeleteLaunchConfigurationInput{
		LaunchConfigurationName: name,
	})
	return errors.WithStackTrace(err)
}
