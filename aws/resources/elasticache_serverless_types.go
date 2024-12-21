package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type ElasticCacheServerlessAPI interface {
	DeleteServerlessCache(ctx context.Context, params *elasticache.DeleteServerlessCacheInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteServerlessCacheOutput, error)
	DescribeServerlessCaches(ctx context.Context, params *elasticache.DescribeServerlessCachesInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeServerlessCachesOutput, error)
}

type ElasticCacheServerless struct {
	BaseAwsResource
	Client     ElasticCacheServerlessAPI
	Region     string
	ClusterIds []string
}

func (cache *ElasticCacheServerless) InitV2(cfg aws.Config) {
	cache.Client = elasticache.NewFromConfig(cfg)
}

func (cache *ElasticCacheServerless) ResourceName() string {
	return "elasticcache-serverless"
}

func (cache *ElasticCacheServerless) ResourceIdentifiers() []string {
	return cache.ClusterIds
}

func (cache *ElasticCacheServerless) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (cache *ElasticCacheServerless) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.Elasticache
}

func (cache *ElasticCacheServerless) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := cache.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	cache.ClusterIds = aws.ToStringSlice(identifiers)
	return cache.ClusterIds, nil
}

func (cache *ElasticCacheServerless) Nuke(identifiers []string) error {
	if err := cache.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
