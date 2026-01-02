package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// SageMakerEndpointConfigAPI defines the interface for SageMaker Endpoint Config operations.
type SageMakerEndpointConfigAPI interface {
	ListEndpointConfigs(ctx context.Context, params *sagemaker.ListEndpointConfigsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListEndpointConfigsOutput, error)
	DeleteEndpointConfig(ctx context.Context, params *sagemaker.DeleteEndpointConfigInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteEndpointConfigOutput, error)
	ListTags(ctx context.Context, params *sagemaker.ListTagsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListTagsOutput, error)
}

// NewSageMakerEndpointConfig creates a new SageMaker Endpoint Config resource using the generic resource pattern.
func NewSageMakerEndpointConfig() AwsResource {
	return NewAwsResource(&resource.Resource[SageMakerEndpointConfigAPI]{
		ResourceTypeName: "sagemaker-endpoint-config",
		BatchSize:        10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[SageMakerEndpointConfigAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = sagemaker.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.SageMakerEndpointConfig
		},
		Lister: listSageMakerEndpointConfigs,
		Nuker:  resource.SimpleBatchDeleter(deleteSageMakerEndpointConfig),
	})
}

// listSageMakerEndpointConfigs retrieves all SageMaker Endpoint Configurations that match the config filters.
func listSageMakerEndpointConfigs(ctx context.Context, client SageMakerEndpointConfigAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	// Get account ID from context (needed for constructing ARN for tags)
	accountID, ok := ctx.Value(util.AccountIdKey).(string)
	if !ok {
		return nil, fmt.Errorf("unable to get account ID from context")
	}

	var endpointConfigNames []*string
	paginator := sagemaker.NewListEndpointConfigsPaginator(client, &sagemaker.ListEndpointConfigsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, endpointConfig := range page.EndpointConfigs {
			if endpointConfig.EndpointConfigName == nil {
				continue
			}

			// Construct the proper ARN for the endpoint config to get tags
			endpointConfigArn := fmt.Sprintf("arn:aws:sagemaker:%s:%s:endpoint-config/%s",
				scope.Region, accountID, *endpointConfig.EndpointConfigName)

			// Get tags for the endpoint config
			tagsOutput, err := client.ListTags(ctx, &sagemaker.ListTagsInput{
				ResourceArn: aws.String(endpointConfigArn),
			})
			if err != nil {
				logging.Debugf("Failed to get tags for endpoint config %s: %s", *endpointConfig.EndpointConfigName, err)
				continue
			}

			tagMap := util.ConvertSageMakerTagsToMap(tagsOutput.Tags)

			if cfg.ShouldInclude(config.ResourceValue{
				Name: endpointConfig.EndpointConfigName,
				Time: endpointConfig.CreationTime,
				Tags: tagMap,
			}) {
				endpointConfigNames = append(endpointConfigNames, endpointConfig.EndpointConfigName)
			}
		}
	}

	return endpointConfigNames, nil
}

// deleteSageMakerEndpointConfig deletes a single SageMaker Endpoint Configuration.
func deleteSageMakerEndpointConfig(ctx context.Context, client SageMakerEndpointConfigAPI, id *string) error {
	_, err := client.DeleteEndpointConfig(ctx, &sagemaker.DeleteEndpointConfigInput{
		EndpointConfigName: id,
	})
	return err
}
