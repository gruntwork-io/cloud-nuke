package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of Elasticache cluster Ids
func getAllElasticacheClusters(session *session.Session, region string, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := elasticache.New(session)

	// First, get any cache clusters that are replication groups, which will be the case for all multi-node Redis clusters
	replicationGroupsResult, replicationGroupsErr := svc.DescribeReplicationGroups(&elasticache.DescribeReplicationGroupsInput{})
	if replicationGroupsErr != nil {
		return nil, errors.WithStackTrace(replicationGroupsErr)
	}

	// Next, get any cache clusters that are not members of a replication group: meaning:
	// 1. any cache clusters with a Engine of "memcached"
	// 2. any single node Redis clusters
	cacheClustersResult, cacheClustersErr := svc.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{
		ShowCacheClustersNotInReplicationGroups: aws.Bool(true),
	})
	if cacheClustersErr != nil {
		return nil, errors.WithStackTrace(cacheClustersErr)
	}

	var clusterIds []*string
	for _, replicationGroup := range replicationGroupsResult.ReplicationGroups {
		if shouldIncludeElasticacheReplicationGroup(replicationGroup, excludeAfter, configObj) {
			clusterIds = append(clusterIds, replicationGroup.ReplicationGroupId)
		}
	}

	for _, cluster := range cacheClustersResult.CacheClusters {
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

func shouldIncludeElasticacheReplicationGroup(replicationGroup *elasticache.ReplicationGroup, excludeAfter time.Time, configObj config.Config) bool {
	if replicationGroup == nil {
		return false
	}

	if excludeAfter.Before(aws.TimeValue(replicationGroup.ReplicationGroupCreateTime)) {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(replicationGroup.ReplicationGroupId),
		configObj.Elasticache.IncludeRule.NamesRegExp,
		configObj.Elasticache.ExcludeRule.NamesRegExp,
	)
}

type CacheClusterType string

const (
	Replication CacheClusterType = "replication"
	Single      CacheClusterType = "single"
)

func determineCacheClusterType(svc *elasticache.ElastiCache, clusterId *string) (*string, CacheClusterType, error) {
	replicationGroupDescribeParams := &elasticache.DescribeReplicationGroupsInput{
		ReplicationGroupId: clusterId,
	}

	replicationGroupOutput, describeReplicationGroupsErr := svc.DescribeReplicationGroups(replicationGroupDescribeParams)
	if describeReplicationGroupsErr != nil {
		if awsErr, ok := describeReplicationGroupsErr.(awserr.Error); ok {
			if awsErr.Code() == elasticache.ErrCodeReplicationGroupNotFoundFault {
				// It's possible that we're looking at a cache cluster, in which case we can safely ignore this error
			} else {
				return nil, Single, describeReplicationGroupsErr
			}
		}
	}

	if len(replicationGroupOutput.ReplicationGroups) > 0 {
		return replicationGroupOutput.ReplicationGroups[0].ReplicationGroupId, Replication, nil
	}

	// A single cache cluster can either be standalone, in the case where Engine is memcached,
	// or a member of a replication group, in the case where Engine is Redis, so we must
	// check the current clusterId via both describe methods, otherwise we'll fail to find it
	describeParams := &elasticache.DescribeCacheClustersInput{
		CacheClusterId: clusterId,
	}

	cacheClustersOutput, describeErr := svc.DescribeCacheClusters(describeParams)
	if describeErr != nil {
		if awsErr, ok := describeErr.(awserr.Error); ok {
			if awsErr.Code() == elasticache.ErrCodeCacheClusterNotFoundFault {
				// It's possible that we're looking at a replication group, in which case we can safely ignore this error
			} else {
				return nil, Single, describeErr
			}
		}
	}

	if len(cacheClustersOutput.CacheClusters) == 1 {
		return cacheClustersOutput.CacheClusters[0].CacheClusterId, Single, nil
	}

	return nil, Single, CouldNotLookupCacheClusterErr{ClusterId: clusterId}
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

func nukeReplicationGroupMemberElasticacheCluster(svc *elasticache.ElastiCache, clusterId *string) error {
	logging.Logger.Infof("Elasticache cluster Id: %s is a member of a replication group. Therefore, deleting its replication group", aws.StringValue(clusterId))

	params := &elasticache.DeleteReplicationGroupInput{
		ReplicationGroupId: clusterId,
	}
	_, err := svc.DeleteReplicationGroup(params)
	if err != nil {
		return err
	}

	waitErr := svc.WaitUntilReplicationGroupDeleted(&elasticache.DescribeReplicationGroupsInput{
		ReplicationGroupId: clusterId,
	})

	if waitErr != nil {
		return waitErr
	}

	logging.Logger.Infof("Successfully deleted replication group Id: %s", aws.StringValue(clusterId))

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
		// We need to look up the cache cluster again to determine if it is a member of a replication group or not,
		// because there are two separate codepaths for deleting a cluster. Cache clusters that are not members of a
		// replication group can be deleted via DeleteCacheCluster, whereas those that are members require a call to
		// DeleteReplicationGroup, which will destroy both the replication group and its member clusters
		clusterId, clusterType, describeErr := determineCacheClusterType(svc, clusterId)
		if describeErr != nil {
			return describeErr
		}

		var err error
		if clusterType == Single {
			err = nukeNonReplicationGroupElasticacheCluster(svc, clusterId)
		} else if clusterType == Replication {
			err = nukeReplicationGroupMemberElasticacheCluster(svc, clusterId)
		}

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(clusterId),
			ResourceType: "Elasticache",
			Error:        err,
		}
		report.Record(e)

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

// Custom errors

type CouldNotLookupCacheClusterErr struct {
	ClusterId *string
}

func (err CouldNotLookupCacheClusterErr) Error() string {
	return fmt.Sprintf("Failed to lookup clusterId: %s", aws.StringValue(err.ClusterId))
}
