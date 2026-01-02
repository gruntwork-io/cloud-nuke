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

// ElasticCacheServerlessAPI defines the interface for ElastiCache Serverless operations.
type ElasticCacheServerlessAPI interface {
	DeleteServerlessCache(ctx context.Context, params *elasticache.DeleteServerlessCacheInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteServerlessCacheOutput, error)
	DescribeServerlessCaches(ctx context.Context, params *elasticache.DescribeServerlessCachesInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeServerlessCachesOutput, error)
}

// NewElasticCacheServerless creates a new ElastiCache Serverless resource using the generic resource pattern.
func NewElasticCacheServerless() AwsResource {
	return NewAwsResource(&resource.Resource[ElasticCacheServerlessAPI]{
		ResourceTypeName: "elasticcache-serverless",
		// Tentative batch size to ensure AWS doesn't throttle
		BatchSize: 49,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[ElasticCacheServerlessAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = elasticache.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ElasticCacheServerless
		},
		Lister: listElasticCacheServerless,
		Nuker:  resource.SimpleBatchDeleter(deleteElasticCacheServerless),
	})
}

// listElasticCacheServerless retrieves all ElastiCache Serverless clusters that match the config filters.
func listElasticCacheServerless(ctx context.Context, client ElasticCacheServerlessAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var output []*string

	paginator := elasticache.NewDescribeServerlessCachesPaginator(client, &elasticache.DescribeServerlessCachesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, cluster := range page.ServerlessCaches {
			if strings.ToLower(*cluster.Status) != "available" {
				continue
			}

			split := strings.Split(*cluster.ARN, ":")
			if len(split) == 0 {
				continue
			}

			name := split[len(split)-1]

			if cfg.ShouldInclude(config.ResourceValue{
				Name: aws.String(name),
				Time: cluster.CreateTime,
			}) {
				output = append(output, aws.String(name))
			}
		}
	}

	return output, nil
}

// deleteElasticCacheServerless deletes a single ElastiCache Serverless cluster.
func deleteElasticCacheServerless(ctx context.Context, client ElasticCacheServerlessAPI, name *string) error {
	_, err := client.DeleteServerlessCache(ctx, &elasticache.DeleteServerlessCacheInput{
		ServerlessCacheName: name,
	})
	return err
}
