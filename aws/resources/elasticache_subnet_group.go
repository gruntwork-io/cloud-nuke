package resources

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// ElasticacheSubnetGroupsAPI defines the interface for Elasticache Subnet Group operations.
type ElasticacheSubnetGroupsAPI interface {
	DescribeCacheSubnetGroups(ctx context.Context, params *elasticache.DescribeCacheSubnetGroupsInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheSubnetGroupsOutput, error)
	DeleteCacheSubnetGroup(ctx context.Context, params *elasticache.DeleteCacheSubnetGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteCacheSubnetGroupOutput, error)
}

// NewElasticacheSubnetGroups creates a new Elasticache Subnet Groups resource using the generic resource pattern.
func NewElasticacheSubnetGroups() AwsResource {
	return NewAwsResource(&resource.Resource[ElasticacheSubnetGroupsAPI]{
		ResourceTypeName: "elasticacheSubnetGroups",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[ElasticacheSubnetGroupsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = elasticache.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ElasticacheSubnetGroups
		},
		Lister: listElasticacheSubnetGroups,
		Nuker:  resource.SimpleBatchDeleter(deleteElasticacheSubnetGroup),
	})
}

// listElasticacheSubnetGroups retrieves all Elasticache subnet groups that match the config filters.
func listElasticacheSubnetGroups(ctx context.Context, client ElasticacheSubnetGroupsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var subnetGroupNames []*string

	paginator := elasticache.NewDescribeCacheSubnetGroupsPaginator(client, &elasticache.DescribeCacheSubnetGroupsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, subnetGroup := range page.CacheSubnetGroups {
			if !strings.Contains(aws.ToString(subnetGroup.CacheSubnetGroupName), "default") &&
				cfg.ShouldInclude(config.ResourceValue{
					Name: subnetGroup.CacheSubnetGroupName,
				}) {
				subnetGroupNames = append(subnetGroupNames, subnetGroup.CacheSubnetGroupName)
			}
		}
	}

	return subnetGroupNames, nil
}

// deleteElasticacheSubnetGroup deletes a single Elasticache subnet group.
func deleteElasticacheSubnetGroup(ctx context.Context, client ElasticacheSubnetGroupsAPI, identifier *string) error {
	_, err := client.DeleteCacheSubnetGroup(ctx, &elasticache.DeleteCacheSubnetGroupInput{
		CacheSubnetGroupName: identifier,
	})
	return err
}
