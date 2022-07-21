package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of Elasticache cluster Ids
func getAllElasticacheClusters(session *session.Session, region string, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := elasticache.New(session)
	result, err := svc.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var clusterIds []*string
	for _, cluster := range result.CacheClusters {
		if shouldIncludeElasticacheCluster(cluster, excludeAfter, configObj) {
			clusterIds = append(clusterIds, cluster.CacheClusterId)
		}
	}

	return clusterIds, nil
}

func shouldIncludeElasticacheCluster(cluster *elasticache.CacheCluster, excludeAfter time.Time, configObj config.Config) bool {
	if cluster == nil {
		return false
	}

	if excludeAfter.Before(aws.TimeValue(cluster.CacheClusterCreateTime)) {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(cluster.CacheClusterId),
		configObj.Elasticache.IncludeRule.NamesRegExp,
		configObj.Elasticache.ExcludeRule.NamesRegExp,
	)
}

func getSingleCacheCluster(svc *elasticache.ElastiCache, clusterId *string) (*elasticache.CacheCluster, error) {
	var cacheCluster *elasticache.CacheCluster

	describeParams := &elasticache.DescribeCacheClustersInput{
		CacheClusterId: clusterId,
	}

	output, describeErr := svc.DescribeCacheClusters(describeParams)
	if describeErr != nil {
		return nil, describeErr
	}

	if len(output.CacheClusters) == 1 {
		cacheCluster = output.CacheClusters[0]
	}
	return cacheCluster, nil
}

func nukeNonReplicationGroupElasticacheCluster(svc *elasticache.ElastiCache, clusterId *string) error {
	logging.Logger.Infof("Deleting Elasticache cluster Id: %s which is not a member of a replication group", aws.StringValue(clusterId))
	params := elasticache.DeleteCacheClusterInput{
		CacheClusterId: clusterId,
	}
	_, err := svc.DeleteCacheCluster(&params)
	if err != nil {
		return err
	}

	return svc.WaitUntilCacheClusterDeleted(&elasticache.DescribeCacheClustersInput{
		CacheClusterId: clusterId,
	})
}

func nukeReplicationGroupMemberElasticacheCluster(svc *elasticache.ElastiCache, cacheCluster *elasticache.CacheCluster) error {
	clusterId := cacheCluster.CacheClusterId
	replicationGroupId := cacheCluster.ReplicationGroupId

	logging.Logger.Infof("Elasticache cluster Id: %s is a member of a replication group. Therefore, deleting its replication group Id: %s", aws.StringValue(clusterId), aws.StringValue(replicationGroupId))

	params := &elasticache.DeleteReplicationGroupInput{
		ReplicationGroupId: replicationGroupId,
	}
	_, err := svc.DeleteReplicationGroup(params)
	if err != nil {
		return err
	}

	waitErr := svc.WaitUntilReplicationGroupDeleted(&elasticache.DescribeReplicationGroupsInput{
		ReplicationGroupId: replicationGroupId,
	})

	if waitErr != nil {
		return waitErr
	}

	logging.Logger.Infof("Successfully deleted replication group Id: %s", replicationGroupId)

	return nil
}

func nukeAllElasticacheClusters(session *session.Session, clusterIds []*string) error {
	svc := elasticache.New(session)

	if len(clusterIds) == 0 {
		logging.Logger.Infof("No Elasticache clusters to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting %d Elasticache clusters in region %s", len(clusterIds), *session.Config.Region)

	var deletedClusterIds []*string
	for _, clusterId := range clusterIds {
		// First, we need to look up the cache cluster again to determine if it is a member of a replication group or not,
		// because members of a replication group require that the replication group be deleted first
		cacheCluster, describeErr := getSingleCacheCluster(svc, clusterId)
		if describeErr != nil {
			return describeErr
		}

		var err error
		if cacheCluster.ReplicationGroupId == nil {
			err = nukeNonReplicationGroupElasticacheCluster(svc, clusterId)
		} else {
			err = nukeReplicationGroupMemberElasticacheCluster(svc, cacheCluster)
		}

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			deletedClusterIds = append(deletedClusterIds, clusterId)
			logging.Logger.Infof("Deleted Elasticache cluster: %s", *clusterId)
		}
	}

	logging.Logger.Infof("[OK] %d Elasticache clusters deleted in %s", len(deletedClusterIds), *session.Config.Region)
	return nil
}
