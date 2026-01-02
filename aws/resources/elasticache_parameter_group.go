package resources

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// ElasticacheParameterGroupsAPI defines the interface for Elasticache Parameter Group operations.
type ElasticacheParameterGroupsAPI interface {
	DescribeCacheParameterGroups(ctx context.Context, params *elasticache.DescribeCacheParameterGroupsInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheParameterGroupsOutput, error)
	DeleteCacheParameterGroup(ctx context.Context, params *elasticache.DeleteCacheParameterGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteCacheParameterGroupOutput, error)
}

// NewElasticacheParameterGroups creates a new Elasticache Parameter Groups resource using the generic resource pattern.
func NewElasticacheParameterGroups() AwsResource {
	return NewAwsResource(&resource.Resource[ElasticacheParameterGroupsAPI]{
		ResourceTypeName: "elasticacheParameterGroups",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[ElasticacheParameterGroupsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = elasticache.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ElasticacheParameterGroups
		},
		Lister: listElasticacheParameterGroups,
		Nuker:  resource.SimpleBatchDeleter(deleteElasticacheParameterGroup),
	})
}

// listElasticacheParameterGroups retrieves all Elasticache parameter groups that match the config filters.
func listElasticacheParameterGroups(ctx context.Context, client ElasticacheParameterGroupsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var paramGroupNames []*string

	paginator := elasticache.NewDescribeCacheParameterGroupsPaginator(client, &elasticache.DescribeCacheParameterGroupsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, paramGroup := range page.CacheParameterGroups {
			if shouldIncludeElasticacheParameterGroup(&paramGroup, cfg) {
				paramGroupNames = append(paramGroupNames, paramGroup.CacheParameterGroupName)
			}
		}
	}

	return paramGroupNames, nil
}

func shouldIncludeElasticacheParameterGroup(paramGroup *types.CacheParameterGroup, cfg config.ResourceType) bool {
	if paramGroup == nil {
		return false
	}
	// Exclude AWS managed resources. user defined resources are unable to begin with "default."
	if strings.HasPrefix(aws.ToString(paramGroup.CacheParameterGroupName), "default.") {
		return false
	}

	return cfg.ShouldInclude(config.ResourceValue{
		Name: paramGroup.CacheParameterGroupName,
	})
}

// deleteElasticacheParameterGroup deletes a single Elasticache parameter group.
func deleteElasticacheParameterGroup(ctx context.Context, client ElasticacheParameterGroupsAPI, identifier *string) error {
	_, err := client.DeleteCacheParameterGroup(ctx, &elasticache.DeleteCacheParameterGroupInput{
		CacheParameterGroupName: identifier,
	})
	return err
}
