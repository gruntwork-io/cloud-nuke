package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ram"
	"github.com/aws/aws-sdk-go-v2/service/ram/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// RAMResourceShareAPI defines the interface for RAM Resource Share operations.
type RAMResourceShareAPI interface {
	GetResourceShares(ctx context.Context, params *ram.GetResourceSharesInput, optFns ...func(*ram.Options)) (*ram.GetResourceSharesOutput, error)
	DeleteResourceShare(ctx context.Context, params *ram.DeleteResourceShareInput, optFns ...func(*ram.Options)) (*ram.DeleteResourceShareOutput, error)
}

// NewResourceShares creates a new RAM Resource Share resource using the generic resource pattern.
func NewResourceShares() AwsResource {
	return NewAwsResource(&resource.Resource[RAMResourceShareAPI]{
		ResourceTypeName: "resource-share",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[RAMResourceShareAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ram.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ResourceShare
		},
		Lister: listResourceShares,
		Nuker:  resource.SimpleBatchDeleter(deleteResourceShare),
	})
}

// listResourceShares retrieves all RAM Resource Shares that match the config filters.
func listResourceShares(ctx context.Context, client RAMResourceShareAPI, _ resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string
	paginator := ram.NewGetResourceSharesPaginator(client, &ram.GetResourceSharesInput{ResourceOwner: types.ResourceOwnerSelf})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, resourceShare := range page.ResourceShares {
			if resourceShare.Status != types.ResourceShareStatusActive {
				continue
			}
			if cfg.ShouldInclude(config.ResourceValue{
				Name: resourceShare.Name,
				Time: resourceShare.CreationTime,
				Tags: convertRAMTagsToMap(resourceShare.Tags),
			}) {
				identifiers = append(identifiers, resourceShare.ResourceShareArn)
			}
		}
	}

	return identifiers, nil
}

// deleteResourceShare deletes a single RAM Resource Share.
func deleteResourceShare(ctx context.Context, client RAMResourceShareAPI, arn *string) error {
	_, err := client.DeleteResourceShare(ctx, &ram.DeleteResourceShareInput{
		ResourceShareArn: arn,
	})
	return err
}

func convertRAMTagsToMap(tags []types.Tag) map[string]string {
	tagMap := make(map[string]string)
	for _, tag := range tags {
		if tag.Key == nil || tag.Value == nil {
			continue
		}
		tagMap[*tag.Key] = *tag.Value
	}

	return tagMap
}
