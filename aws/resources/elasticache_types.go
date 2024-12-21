package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type ElasticachesAPI interface {
	DescribeReplicationGroups(ctx context.Context, params *elasticache.DescribeReplicationGroupsInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeReplicationGroupsOutput, error)
	DescribeCacheClusters(ctx context.Context, params *elasticache.DescribeCacheClustersInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheClustersOutput, error)
	DeleteCacheCluster(ctx context.Context, params *elasticache.DeleteCacheClusterInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteCacheClusterOutput, error)
	DeleteReplicationGroup(ctx context.Context, params *elasticache.DeleteReplicationGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteReplicationGroupOutput, error)
}

// Elasticaches - represents all Elasticache clusters
type Elasticaches struct {
	BaseAwsResource
	Client     ElasticachesAPI
	Region     string
	ClusterIds []string
}

func (cache *Elasticaches) InitV2(cfg aws.Config) {
	cache.Client = elasticache.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (cache *Elasticaches) ResourceName() string {
	return "elasticache"
}

// ResourceIdentifiers - The instance ids of the elasticache clusters
func (cache *Elasticaches) ResourceIdentifiers() []string {
	return cache.ClusterIds
}

func (cache *Elasticaches) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (cache *Elasticaches) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.Elasticache
}

func (cache *Elasticaches) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := cache.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	cache.ClusterIds = aws.ToStringSlice(identifiers)
	return cache.ClusterIds, nil
}

// Nuke - nuke 'em all!!!
func (cache *Elasticaches) Nuke(identifiers []string) error {
	if err := cache.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

/*
Elasticache Parameter Groups
*/
type ElasticacheParameterGroupsAPI interface {
	DescribeCacheParameterGroups(ctx context.Context, params *elasticache.DescribeCacheParameterGroupsInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheParameterGroupsOutput, error)
	DeleteCacheParameterGroup(ctx context.Context, params *elasticache.DeleteCacheParameterGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteCacheParameterGroupOutput, error)
}

type ElasticacheParameterGroups struct {
	BaseAwsResource
	Client     ElasticacheParameterGroupsAPI
	Region     string
	GroupNames []string
}

func (pg *ElasticacheParameterGroups) InitV2(cfg aws.Config) {
	pg.Client = elasticache.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (pg *ElasticacheParameterGroups) ResourceName() string {
	return "elasticacheParameterGroups"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (pg *ElasticacheParameterGroups) ResourceIdentifiers() []string {
	return pg.GroupNames
}

func (pg *ElasticacheParameterGroups) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (pg *ElasticacheParameterGroups) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := pg.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	pg.GroupNames = aws.ToStringSlice(identifiers)
	return pg.GroupNames, nil
}

// Nuke - nuke 'em all!!!
func (pg *ElasticacheParameterGroups) Nuke(identifiers []string) error {
	if err := pg.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

/*
Elasticache Subnet Groups
*/
type ElasticacheSubnetGroupsAPI interface {
	DescribeCacheSubnetGroups(ctx context.Context, params *elasticache.DescribeCacheSubnetGroupsInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheSubnetGroupsOutput, error)
	DeleteCacheSubnetGroup(ctx context.Context, params *elasticache.DeleteCacheSubnetGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteCacheSubnetGroupOutput, error)
}

type ElasticacheSubnetGroups struct {
	BaseAwsResource
	Client     ElasticacheSubnetGroupsAPI
	Region     string
	GroupNames []string
}

func (sg *ElasticacheSubnetGroups) InitV2(cfg aws.Config) {
	sg.Client = elasticache.NewFromConfig(cfg)
}

func (sg *ElasticacheSubnetGroups) ResourceName() string {
	return "elasticacheSubnetGroups"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (sg *ElasticacheSubnetGroups) ResourceIdentifiers() []string {
	return sg.GroupNames
}

func (sg *ElasticacheSubnetGroups) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (sg *ElasticacheSubnetGroups) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := sg.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	sg.GroupNames = aws.ToStringSlice(identifiers)
	return sg.GroupNames, nil
}

// Nuke - nuke 'em all!!!
func (sg *ElasticacheSubnetGroups) Nuke(identifiers []string) error {
	if err := sg.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
