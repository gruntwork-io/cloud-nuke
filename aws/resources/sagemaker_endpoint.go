package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// SageMakerEndpointAPI defines the interface for SageMaker Endpoint operations.
type SageMakerEndpointAPI interface {
	ListEndpoints(ctx context.Context, params *sagemaker.ListEndpointsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListEndpointsOutput, error)
	DeleteEndpoint(ctx context.Context, params *sagemaker.DeleteEndpointInput, optFns ...func(*sagemaker.Options)) (*sagemaker.DeleteEndpointOutput, error)
	ListTags(ctx context.Context, params *sagemaker.ListTagsInput, optFns ...func(*sagemaker.Options)) (*sagemaker.ListTagsOutput, error)
}

// NewSageMakerEndpoint creates a new SageMakerEndpoint resource using the generic resource pattern.
func NewSageMakerEndpoint() AwsResource {
	return NewAwsResource(&resource.Resource[SageMakerEndpointAPI]{
		ResourceTypeName: "sagemaker-endpoint",
		BatchSize:        10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[SageMakerEndpointAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = sagemaker.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.SageMakerEndpoint
		},
		Lister: listSageMakerEndpoints,
		Nuker:  resource.SimpleBatchDeleter(deleteSageMakerEndpoint),
	})
}

// listSageMakerEndpoints retrieves all SageMaker Endpoints that match the config filters.
func listSageMakerEndpoints(ctx context.Context, client SageMakerEndpointAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var endpointNames []*string
	paginator := sagemaker.NewListEndpointsPaginator(client, &sagemaker.ListEndpointsInput{})

	// Get account ID from context
	accountID, ok := ctx.Value(util.AccountIdKey).(string)
	if !ok {
		return nil, errors.WithStackTrace(fmt.Errorf("unable to get account ID from context"))
	}

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, endpoint := range output.Endpoints {
			if endpoint.EndpointName == nil {
				continue
			}

			logging.Debugf("Found SageMaker Endpoint: %s (Status: %s)", *endpoint.EndpointName, endpoint.EndpointStatus)

			// Construct the proper ARN for the endpoint
			endpointArn := fmt.Sprintf("arn:aws:sagemaker:%s:%s:endpoint/%s",
				scope.Region, accountID, *endpoint.EndpointName)

			// Get tags for the endpoint
			tagsOutput, err := client.ListTags(ctx, &sagemaker.ListTagsInput{
				ResourceArn: aws.String(endpointArn),
			})
			if err != nil {
				logging.Debugf("Failed to get tags for endpoint %s: %s", *endpoint.EndpointName, err)
				continue
			}

			tagMap := util.ConvertSageMakerTagsToMap(tagsOutput.Tags)

			// Check tag exclusion rules
			if shouldExcludeByTags(endpoint.EndpointName, tagMap, cfg) {
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: endpoint.EndpointName,
				Time: endpoint.CreationTime,
				Tags: tagMap,
			}) {
				endpointNames = append(endpointNames, endpoint.EndpointName)
			}
		}
	}

	return endpointNames, nil
}

// shouldExcludeByTags checks if the endpoint should be excluded based on tag rules.
func shouldExcludeByTags(endpointName *string, tagMap map[string]string, cfg config.ResourceType) bool {
	// Check the newer Tags map approach
	for tag, pattern := range cfg.ExcludeRule.Tags {
		if tagValue, hasTag := tagMap[tag]; hasTag {
			if pattern.RE.MatchString(tagValue) {
				logging.Debugf("Excluding endpoint %s due to tag '%s' with value '%s' matching pattern '%s'",
					*endpointName, tag, tagValue, pattern.RE.String())
				return true
			}
		}
	}

	// Check the deprecated Tag/TagValue approach
	if cfg.ExcludeRule.Tag != nil {
		tagName := *cfg.ExcludeRule.Tag
		if tagValue, hasTag := tagMap[tagName]; hasTag {
			if cfg.ExcludeRule.TagValue != nil {
				if cfg.ExcludeRule.TagValue.RE.MatchString(tagValue) {
					logging.Debugf("Excluding endpoint %s due to deprecated tag '%s' with value '%s' matching pattern '%s'",
						*endpointName, tagName, tagValue, cfg.ExcludeRule.TagValue.RE.String())
					return true
				}
			} else if strings.EqualFold(tagValue, "true") {
				logging.Debugf("Excluding endpoint %s due to deprecated tag '%s' with default value 'true'",
					*endpointName, tagName)
				return true
			}
		}
	}

	return false
}

// deleteSageMakerEndpoint deletes a single SageMaker Endpoint.
func deleteSageMakerEndpoint(ctx context.Context, client SageMakerEndpointAPI, endpointName *string) error {
	logging.Debugf("Deleting SageMaker Endpoint: %s", aws.ToString(endpointName))

	_, err := client.DeleteEndpoint(ctx, &sagemaker.DeleteEndpointInput{
		EndpointName: endpointName,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
