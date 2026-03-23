package resources

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// SSMParameterAPI defines the interface for SSM Parameter Store operations.
type SSMParameterAPI interface {
	DescribeParameters(ctx context.Context, params *ssm.DescribeParametersInput, optFns ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error)
	ListTagsForResource(ctx context.Context, params *ssm.ListTagsForResourceInput, optFns ...func(*ssm.Options)) (*ssm.ListTagsForResourceOutput, error)
	DeleteParameter(ctx context.Context, params *ssm.DeleteParameterInput, optFns ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error)
}

// NewSSMParameter creates a new SSMParameter resource using the generic resource pattern.
func NewSSMParameter() AwsResource {
	return NewAwsResource(&resource.Resource[SSMParameterAPI]{
		ResourceTypeName: "ssm-parameter",
		// Conservative batch size to avoid AWS throttling.
		BatchSize: 10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[SSMParameterAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ssm.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.SSMParameter
		},
		Lister: listSSMParameters,
		Nuker:  resource.SimpleBatchDeleter(deleteSSMParameter),
	})
}

// listSSMParameters retrieves all SSM parameters that match the config filters.
func listSSMParameters(ctx context.Context, client SSMParameterAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var names []*string

	paginator := ssm.NewDescribeParametersPaginator(client, &ssm.DescribeParametersInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, param := range page.Parameters {
			if param.Name == nil {
				continue
			}

			// AWS reserves the /aws/ prefix for public parameters (e.g. /aws/service/*, /aws/reference/*).
			// Customers cannot create parameters under this path, and deletion will fail with AccessDeniedException.
			if strings.HasPrefix(aws.ToString(param.Name), "/aws/") {
				logging.Debugf("Skipping %s since it is an AWS-managed parameter", aws.ToString(param.Name))
				continue
			}

			tags, err := getSSMParameterTags(ctx, client, param.Name)
			if err != nil {
				// Skip rather than proceed with nil tags. Passing nil to ShouldInclude
				// signals "no tag support", which bypasses cloud-nuke-excluded and
				// cloud-nuke-after protection checks. Skip this run; retry on the next.
				logging.Errorf("Unable to fetch tags for SSM parameter %s: %s", aws.ToString(param.Name), err)
				continue
			}

			// SSM DescribeParameters does not expose a creation time; LastModifiedDate is the
			// closest available timestamp and is used as the time filter for this resource type.
			if cfg.ShouldInclude(config.ResourceValue{
				Name: param.Name,
				Time: param.LastModifiedDate,
				Tags: tags,
			}) {
				names = append(names, param.Name)
			}
		}
	}

	return names, nil
}

// getSSMParameterTags returns the tags for an SSM parameter by name.
// Returns an empty non-nil map when the parameter has no tags.
func getSSMParameterTags(ctx context.Context, client SSMParameterAPI, name *string) (map[string]string, error) {
	output, err := client.ListTagsForResource(ctx, &ssm.ListTagsForResourceInput{
		ResourceId:   name,
		ResourceType: types.ResourceTypeForTaggingParameter,
	})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	tags := make(map[string]string, len(output.TagList))
	for _, tag := range output.TagList {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}

	return tags, nil
}

// deleteSSMParameter deletes a single SSM parameter by name.
func deleteSSMParameter(ctx context.Context, client SSMParameterAPI, name *string) error {
	_, err := client.DeleteParameter(ctx, &ssm.DeleteParameterInput{
		Name: name,
	})
	return errors.WithStackTrace(err)
}
