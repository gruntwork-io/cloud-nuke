package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// CloudMapNamespacesAPI defines the interface for AWS Cloud Map operations.
type CloudMapNamespacesAPI interface {
	ListNamespaces(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error)
	DeleteNamespace(ctx context.Context, params *servicediscovery.DeleteNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteNamespaceOutput, error)
	GetNamespace(ctx context.Context, params *servicediscovery.GetNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetNamespaceOutput, error)
	ListTagsForResource(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error)
}

// NewCloudMapNamespaces creates a new CloudMapNamespaces resource using the generic resource pattern.
func NewCloudMapNamespaces() AwsResource {
	return NewAwsResource(&resource.Resource[CloudMapNamespacesAPI]{
		ResourceTypeName: "cloudmap-namespace",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[CloudMapNamespacesAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = servicediscovery.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.CloudMapNamespace
		},
		Lister: listCloudMapNamespaces,
		Nuker:  resource.SequentialDeleter(deleteCloudMapNamespace),
	})
}

// listCloudMapNamespaces retrieves all Cloud Map namespaces that match the config filters.
func listCloudMapNamespaces(ctx context.Context, client CloudMapNamespacesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var namespaceIds []*string

	paginator := servicediscovery.NewListNamespacesPaginator(client, &servicediscovery.ListNamespacesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, namespace := range page.Namespaces {
			tags, err := getNamespaceTags(ctx, client, namespace.Arn)
			if err != nil {
				return nil, errors.WithStackTrace(err)
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: namespace.Name,
				Time: namespace.CreateDate,
				Tags: tags,
			}) {
				namespaceIds = append(namespaceIds, namespace.Id)
			}
		}
	}

	return namespaceIds, nil
}

// deleteCloudMapNamespace deletes a single Cloud Map namespace.
// Note: Services must be deleted before namespaces (handled by resource ordering in resource_registry.go).
func deleteCloudMapNamespace(ctx context.Context, client CloudMapNamespacesAPI, id *string) error {
	// Log Route53 hosted zone info if present (will be auto-cleaned by AWS)
	if err := logHostedZoneInfo(ctx, client, id); err != nil {
		logging.Debugf("Error getting namespace %s: %s", aws.ToString(id), err)
	}

	_, err := client.DeleteNamespace(ctx, &servicediscovery.DeleteNamespaceInput{Id: id})
	return err
}

// logHostedZoneInfo logs information about Route53 hosted zones that will be auto-cleaned.
func logHostedZoneInfo(ctx context.Context, client CloudMapNamespacesAPI, id *string) error {
	resp, err := client.GetNamespace(ctx, &servicediscovery.GetNamespaceInput{Id: id})
	if err != nil {
		return err
	}

	if resp.Namespace.Properties != nil &&
		resp.Namespace.Properties.DnsProperties != nil &&
		resp.Namespace.Properties.DnsProperties.HostedZoneId != nil {
		logging.Debugf("Namespace %s has associated Route53 hosted zone %s - it will be cleaned automatically",
			aws.ToString(id), aws.ToString(resp.Namespace.Properties.DnsProperties.HostedZoneId))
	}

	return nil
}

// getNamespaceTags retrieves all tags for a Cloud Map namespace.
func getNamespaceTags(ctx context.Context, client CloudMapNamespacesAPI, arn *string) (map[string]string, error) {
	resp, err := client.ListTagsForResource(ctx, &servicediscovery.ListTagsForResourceInput{
		ResourceARN: arn,
	})
	if err != nil {
		logging.Debugf("Error getting tags for Cloud Map namespace with ARN %s", aws.ToString(arn))
		return nil, errors.WithStackTrace(err)
	}

	tags := make(map[string]string)
	for _, tag := range resp.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}

	return tags, nil
}
