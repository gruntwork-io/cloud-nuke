package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// Route53CidrCollectionAPI defines the interface for Route53 CIDR Collection operations.
type Route53CidrCollectionAPI interface {
	ListCidrCollections(ctx context.Context, params *route53.ListCidrCollectionsInput, optFns ...func(*route53.Options)) (*route53.ListCidrCollectionsOutput, error)
	ListCidrBlocks(ctx context.Context, params *route53.ListCidrBlocksInput, optFns ...func(*route53.Options)) (*route53.ListCidrBlocksOutput, error)
	ChangeCidrCollection(ctx context.Context, params *route53.ChangeCidrCollectionInput, optFns ...func(*route53.Options)) (*route53.ChangeCidrCollectionOutput, error)
	DeleteCidrCollection(ctx context.Context, params *route53.DeleteCidrCollectionInput, optFns ...func(*route53.Options)) (*route53.DeleteCidrCollectionOutput, error)
}

// NewRoute53CidrCollections creates a new Route53 CIDR Collection resource using the generic resource pattern.
// Route53 is a global service.
func NewRoute53CidrCollections() AwsResource {
	return NewAwsResource(&resource.Resource[Route53CidrCollectionAPI]{
		ResourceTypeName: "route53-cidr-collection",
		BatchSize:        DefaultBatchSize,
		IsGlobal:         true,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[Route53CidrCollectionAPI], cfg aws.Config) {
			r.Scope.Region = "global"
			r.Client = route53.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.Route53CIDRCollection
		},
		Lister: listRoute53CidrCollections,
		Nuker:  resource.MultiStepDeleter(nukeCidrBlocks, deleteRoute53CidrCollection),
	})
}

// listRoute53CidrCollections retrieves all CIDR collections that match the config filters.
func listRoute53CidrCollections(ctx context.Context, client Route53CidrCollectionAPI, _ resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	paginator := route53.NewListCidrCollectionsPaginator(client, &route53.ListCidrCollectionsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, collection := range page.CidrCollections {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: collection.Name,
			}) {
				identifiers = append(identifiers, collection.Id)
			}
		}
	}

	return identifiers, nil
}

// nukeCidrBlocks removes all CIDR blocks from a collection before it can be deleted.
func nukeCidrBlocks(ctx context.Context, client Route53CidrCollectionAPI, id *string) error {
	logging.Debugf("Removing CIDR blocks from collection %s", aws.ToString(id))

	var allChanges []types.CidrCollectionChange

	paginator := route53.NewListCidrBlocksPaginator(client, &route53.ListCidrBlocksInput{
		CollectionId: id,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return err
		}

		for _, block := range page.CidrBlocks {
			allChanges = append(allChanges, types.CidrCollectionChange{
				CidrList:     []string{aws.ToString(block.CidrBlock)},
				Action:       types.CidrCollectionChangeActionDeleteIfExists,
				LocationName: block.LocationName,
			})
		}
	}

	if len(allChanges) == 0 {
		return nil
	}

	_, err := client.ChangeCidrCollection(ctx, &route53.ChangeCidrCollectionInput{
		Id:      id,
		Changes: allChanges,
	})
	if err != nil {
		return err
	}

	logging.Debugf("Successfully removed CIDR blocks from collection %s", aws.ToString(id))
	return nil
}

// deleteRoute53CidrCollection deletes a single CIDR collection.
func deleteRoute53CidrCollection(ctx context.Context, client Route53CidrCollectionAPI, id *string) error {
	_, err := client.DeleteCidrCollection(ctx, &route53.DeleteCidrCollectionInput{
		Id: id,
	})
	return err
}
