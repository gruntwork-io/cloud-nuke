package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// LaunchTemplatesAPI defines the interface for Launch Template operations.
type LaunchTemplatesAPI interface {
	DescribeLaunchTemplates(ctx context.Context, params *ec2.DescribeLaunchTemplatesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeLaunchTemplatesOutput, error)
	DeleteLaunchTemplate(ctx context.Context, params *ec2.DeleteLaunchTemplateInput, optFns ...func(*ec2.Options)) (*ec2.DeleteLaunchTemplateOutput, error)
}

// NewLaunchTemplates creates a new Launch Templates resource using the generic resource pattern.
func NewLaunchTemplates() AwsResource {
	return NewAwsResource(&resource.Resource[LaunchTemplatesAPI]{
		ResourceTypeName: "lt",
		BatchSize:        49,
		InitClient: func(r *resource.Resource[LaunchTemplatesAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for EC2 client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = ec2.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.LaunchTemplate
		},
		Lister: listLaunchTemplates,
		Nuker:  resource.SimpleBatchDeleter(deleteLaunchTemplate),
	})
}

// listLaunchTemplates retrieves all launch templates that match the config filters.
func listLaunchTemplates(ctx context.Context, client LaunchTemplatesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var names []*string

	paginator := ec2.NewDescribeLaunchTemplatesPaginator(client, &ec2.DescribeLaunchTemplatesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, template := range page.LaunchTemplates {
			// Extract tags from the launch template
			tags := make(map[string]string)
			for _, tag := range template.Tags {
				if tag.Key != nil && tag.Value != nil {
					tags[*tag.Key] = *tag.Value
				}
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: template.LaunchTemplateName,
				Time: template.CreateTime,
				Tags: tags,
			}) {
				names = append(names, template.LaunchTemplateName)
			}
		}
	}

	return names, nil
}

// deleteLaunchTemplate deletes a single launch template.
func deleteLaunchTemplate(ctx context.Context, client LaunchTemplatesAPI, name *string) error {
	_, err := client.DeleteLaunchTemplate(ctx, &ec2.DeleteLaunchTemplateInput{
		LaunchTemplateName: name,
	})
	return err
}
