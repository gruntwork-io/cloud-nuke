package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// SesConfigurationSetAPI defines the interface for SES Configuration Set operations.
type SesConfigurationSetAPI interface {
	ListConfigurationSets(ctx context.Context, params *ses.ListConfigurationSetsInput, optFns ...func(*ses.Options)) (*ses.ListConfigurationSetsOutput, error)
	DeleteConfigurationSet(ctx context.Context, params *ses.DeleteConfigurationSetInput, optFns ...func(*ses.Options)) (*ses.DeleteConfigurationSetOutput, error)
}

// NewSesConfigurationSet creates a new SES Configuration Set resource using the generic resource pattern.
func NewSesConfigurationSet() AwsResource {
	return NewAwsResource(&resource.Resource[SesConfigurationSetAPI]{
		ResourceTypeName: "ses-configuration-set",
		BatchSize:        maxBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[SesConfigurationSetAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ses.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.SESConfigurationSet
		},
		Lister: listSesConfigurationSets,
		Nuker:  resource.SimpleBatchDeleter(deleteSesConfigurationSet),
	})
}

// listSesConfigurationSets retrieves all SES configuration sets that match the config filters.
func listSesConfigurationSets(ctx context.Context, client SesConfigurationSetAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var configSets []*string
	var nextToken *string

	for {
		result, err := client.ListConfigurationSets(ctx, &ses.ListConfigurationSetsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}

		for _, set := range result.ConfigurationSets {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: set.Name,
			}) {
				configSets = append(configSets, set.Name)
			}
		}

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	return configSets, nil
}

// deleteSesConfigurationSet deletes a single SES configuration set.
func deleteSesConfigurationSet(ctx context.Context, client SesConfigurationSetAPI, configSetName *string) error {
	_, err := client.DeleteConfigurationSet(ctx, &ses.DeleteConfigurationSetInput{
		ConfigurationSetName: configSetName,
	})
	return err
}
