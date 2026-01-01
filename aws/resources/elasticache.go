package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	goerrors "github.com/gruntwork-io/go-commons/errors"
)

// ElasticachesAPI defines the interface for Elasticache operations.
type ElasticachesAPI interface {
	DescribeReplicationGroups(ctx context.Context, params *elasticache.DescribeReplicationGroupsInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeReplicationGroupsOutput, error)
	DescribeCacheClusters(ctx context.Context, params *elasticache.DescribeCacheClustersInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheClustersOutput, error)
	DeleteCacheCluster(ctx context.Context, params *elasticache.DeleteCacheClusterInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteCacheClusterOutput, error)
	DeleteReplicationGroup(ctx context.Context, params *elasticache.DeleteReplicationGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteReplicationGroupOutput, error)
}

// NewElasticaches creates a new Elasticaches resource using the generic resource pattern.
func NewElasticaches() AwsResource {
	return NewAwsResource(&resource.Resource[ElasticachesAPI]{
		ResourceTypeName: "elasticache",
		// Tentative batch size to ensure AWS doesn't throttle
		BatchSize: 49,
		InitClient: func(r *resource.Resource[ElasticachesAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for Elasticache client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = elasticache.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.Elasticache
		},
		Lister: listElasticaches,
		// Use SequentialDeleter since each deletion involves waiters
		Nuker: resource.SequentialDeleter(deleteElasticacheCluster),
	})
}

// listElasticaches retrieves all Elasticache clusters that match the config filters.
func listElasticaches(ctx context.Context, client ElasticachesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	// First, get any cache clusters that are replication groups, which will be the case for all multi-node Redis clusters
	replicationGroupsResult, replicationGroupsErr := client.DescribeReplicationGroups(ctx, &elasticache.DescribeReplicationGroupsInput{})
	if replicationGroupsErr != nil {
		return nil, goerrors.WithStackTrace(replicationGroupsErr)
	}

	// Next, get any cache clusters that are not members of a replication group: meaning:
	// 1. any cache clusters with an Engine of "memcached"
	// 2. any single node Redis clusters
	cacheClustersResult, cacheClustersErr := client.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{
		ShowCacheClustersNotInReplicationGroups: aws.Bool(true),
	})
	if cacheClustersErr != nil {
		return nil, goerrors.WithStackTrace(cacheClustersErr)
	}

	var clusterIds []*string
	for _, replicationGroup := range replicationGroupsResult.ReplicationGroups {
		if cfg.ShouldInclude(config.ResourceValue{
			Name: replicationGroup.ReplicationGroupId,
			Time: replicationGroup.ReplicationGroupCreateTime,
		}) {
			clusterIds = append(clusterIds, replicationGroup.ReplicationGroupId)
		}
	}

	for _, cluster := range cacheClustersResult.CacheClusters {
		if cfg.ShouldInclude(config.ResourceValue{
			Name: cluster.CacheClusterId,
			Time: cluster.CacheClusterCreateTime,
		}) {
			clusterIds = append(clusterIds, cluster.CacheClusterId)
		}
	}

	return clusterIds, nil
}

type CacheClusterType string

const (
	Replication CacheClusterType = "replication"
	Single      CacheClusterType = "single"
)

func determineCacheClusterType(ctx context.Context, client ElasticachesAPI, clusterId *string) (*string, CacheClusterType, error) {
	replicationGroupDescribeParams := &elasticache.DescribeReplicationGroupsInput{
		ReplicationGroupId: clusterId,
	}

	replicationGroupOutput, describeReplicationGroupsErr := client.DescribeReplicationGroups(ctx, replicationGroupDescribeParams)
	if describeReplicationGroupsErr != nil {
		// GlobalReplicationGroupNotFoundFault
		var eRG404 *types.ReplicationGroupNotFoundFault
		if errors.As(describeReplicationGroupsErr, &eRG404) {
			// It's possible that we're looking at a cache cluster, in which case we can safely ignore this error
		} else {
			return nil, Single, describeReplicationGroupsErr
		}
	}

	if replicationGroupOutput != nil && len(replicationGroupOutput.ReplicationGroups) > 0 {
		return replicationGroupOutput.ReplicationGroups[0].ReplicationGroupId, Replication, nil
	}

	// A single cache cluster can either be standalone, in the case where Engine is memcached,
	// or a member of a replication group, in the case where Engine is Redis, so we must
	// check the current clusterId via both describe methods, otherwise we'll fail to find it
	describeParams := &elasticache.DescribeCacheClustersInput{
		CacheClusterId: clusterId,
	}

	cacheClustersOutput, describeErr := client.DescribeCacheClusters(ctx, describeParams)
	if describeErr != nil {
		var eC404 *types.CacheClusterNotFoundFault
		if errors.As(describeErr, &eC404) {
			// It's possible that we're looking at a replication group, in which case we can safely ignore this error
		} else {
			return nil, Single, describeErr
		}
	}

	if len(cacheClustersOutput.CacheClusters) == 1 {
		return cacheClustersOutput.CacheClusters[0].CacheClusterId, Single, nil
	}

	return nil, Single, CouldNotLookupCacheClusterErr{ClusterId: clusterId}
}

func nukeNonReplicationGroupElasticacheCluster(ctx context.Context, client ElasticachesAPI, clusterId *string) error {
	logging.Debugf("Deleting Elasticache cluster Id: %s which is not a member of a replication group", aws.ToString(clusterId))
	params := elasticache.DeleteCacheClusterInput{
		CacheClusterId: clusterId,
	}
	_, err := client.DeleteCacheCluster(ctx, &params)
	if err != nil {
		return err
	}

	waiter := elasticache.NewCacheClusterDeletedWaiter(client)

	return waiter.Wait(ctx, &elasticache.DescribeCacheClustersInput{
		CacheClusterId: clusterId,
	}, DefaultWaitTimeout)
}

func nukeReplicationGroupMemberElasticacheCluster(ctx context.Context, client ElasticachesAPI, clusterId *string) error {
	logging.Debugf("Elasticache cluster Id: %s is a member of a replication group. Therefore, deleting its replication group", aws.ToString(clusterId))

	params := &elasticache.DeleteReplicationGroupInput{
		ReplicationGroupId: clusterId,
	}
	_, err := client.DeleteReplicationGroup(ctx, params)
	if err != nil {
		return err
	}

	waiter := elasticache.NewReplicationGroupDeletedWaiter(client)
	waitErr := waiter.Wait(ctx, &elasticache.DescribeReplicationGroupsInput{ReplicationGroupId: clusterId}, DefaultWaitTimeout)

	if waitErr != nil {
		return waitErr
	}

	logging.Debugf("Successfully deleted replication group Id: %s", aws.ToString(clusterId))

	return nil
}

// deleteElasticacheCluster deletes a single Elasticache cluster.
// It determines whether the cluster is standalone or part of a replication group
// and calls the appropriate delete function.
func deleteElasticacheCluster(ctx context.Context, client ElasticachesAPI, clusterId *string) error {
	// We need to look up the cache cluster to determine if it is a member of a replication group or not,
	// because there are two separate codepaths for deleting a cluster. Cache clusters that are not members of a
	// replication group can be deleted via DeleteCacheCluster, whereas those that are members require a call to
	// DeleteReplicationGroup, which will destroy both the replication group and its member clusters
	resolvedClusterId, clusterType, err := determineCacheClusterType(ctx, client, clusterId)
	if err != nil {
		return err
	}

	if clusterType == Single {
		return nukeNonReplicationGroupElasticacheCluster(ctx, client, resolvedClusterId)
	}
	return nukeReplicationGroupMemberElasticacheCluster(ctx, client, resolvedClusterId)
}

// Custom errors

type CouldNotLookupCacheClusterErr struct {
	ClusterId *string
}

func (err CouldNotLookupCacheClusterErr) Error() string {
	return fmt.Sprintf("Failed to lookup clusterId: %s", aws.ToString(err.ClusterId))
}
