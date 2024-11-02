package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	goerrors "github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of Elasticache cluster Ids
func (cache *Elasticaches) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	// First, get any cache clusters that are replication groups, which will be the case for all multi-node Redis clusters
	replicationGroupsResult, replicationGroupsErr := cache.Client.DescribeReplicationGroups(cache.Context, &elasticache.DescribeReplicationGroupsInput{})
	if replicationGroupsErr != nil {
		return nil, goerrors.WithStackTrace(replicationGroupsErr)
	}

	// Next, get any cache clusters that are not members of a replication group: meaning:
	// 1. any cache clusters with an Engine of "memcached"
	// 2. any single node Redis clusters
	cacheClustersResult, cacheClustersErr := cache.Client.DescribeCacheClusters(
		cache.Context,
		&elasticache.DescribeCacheClustersInput{
			ShowCacheClustersNotInReplicationGroups: aws.Bool(true),
		})
	if cacheClustersErr != nil {
		return nil, goerrors.WithStackTrace(cacheClustersErr)
	}

	var clusterIds []*string
	for _, replicationGroup := range replicationGroupsResult.ReplicationGroups {
		if configObj.Elasticache.ShouldInclude(config.ResourceValue{
			Name: replicationGroup.ReplicationGroupId,
			Time: replicationGroup.ReplicationGroupCreateTime,
		}) {
			clusterIds = append(clusterIds, replicationGroup.ReplicationGroupId)
		}
	}

	for _, cluster := range cacheClustersResult.CacheClusters {
		if configObj.Elasticache.ShouldInclude(config.ResourceValue{
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

func (cache *Elasticaches) determineCacheClusterType(clusterId *string) (*string, CacheClusterType, error) {
	replicationGroupDescribeParams := &elasticache.DescribeReplicationGroupsInput{
		ReplicationGroupId: clusterId,
	}

	replicationGroupOutput, describeReplicationGroupsErr := cache.Client.DescribeReplicationGroups(cache.Context, replicationGroupDescribeParams)
	if describeReplicationGroupsErr != nil {
		// GlobalReplicationGroupNotFoundFault
		var eRG404 *types.ReplicationGroupNotFoundFault
		if errors.As(describeReplicationGroupsErr, &eRG404) {
			// It's possible that we're looking at a cache cluster, in which case we can safely ignore this error
		} else {
			return nil, Single, describeReplicationGroupsErr
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

	cacheClustersOutput, describeErr := cache.Client.DescribeCacheClusters(cache.Context, describeParams)
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

func (cache *Elasticaches) nukeNonReplicationGroupElasticacheCluster(clusterId *string) error {
	logging.Debugf("Deleting Elasticache cluster Id: %s which is not a member of a replication group", aws.ToString(clusterId))
	params := elasticache.DeleteCacheClusterInput{
		CacheClusterId: clusterId,
	}
	_, err := cache.Client.DeleteCacheCluster(cache.Context, &params)
	if err != nil {
		return err
	}

	waiter := elasticache.NewCacheClusterDeletedWaiter(cache.Client)

	return waiter.Wait(
		cache.Context,
		&elasticache.DescribeCacheClustersInput{
			CacheClusterId: clusterId,
		},
		15*time.Minute,
	)
}

func (cache *Elasticaches) nukeReplicationGroupMemberElasticacheCluster(clusterId *string) error {
	logging.Debugf("Elasticache cluster Id: %s is a member of a replication group. Therefore, deleting its replication group", aws.ToString(clusterId))

	params := &elasticache.DeleteReplicationGroupInput{
		ReplicationGroupId: clusterId,
	}
	_, err := cache.Client.DeleteReplicationGroup(cache.Context, params)
	if err != nil {
		return err
	}

	waiter := elasticache.NewReplicationGroupDeletedWaiter(cache.Client)
	waitErr := waiter.Wait(
		cache.Context,
		&elasticache.DescribeReplicationGroupsInput{ReplicationGroupId: clusterId},
		15*time.Minute,
	)

	if waitErr != nil {
		return waitErr
	}

	logging.Debugf("Successfully deleted replication group Id: %s", aws.ToString(clusterId))

	return nil
}

func (cache *Elasticaches) nukeAll(clusterIds []*string) error {
	if len(clusterIds) == 0 {
		logging.Debugf("No Elasticache clusters to nuke in region %s", cache.Region)
		return nil
	}

	logging.Debugf("Deleting %d Elasticache clusters in region %s", len(clusterIds), cache.Region)

	var deletedClusterIds []*string
	for _, clusterId := range clusterIds {
		// We need to look up the cache cluster again to determine if it is a member of a replication group or not,
		// because there are two separate codepaths for deleting a cluster. Cache clusters that are not members of a
		// replication group can be deleted via DeleteCacheCluster, whereas those that are members require a call to
		// DeleteReplicationGroup, which will destroy both the replication group and its member clusters
		clusterId, clusterType, describeErr := cache.determineCacheClusterType(clusterId)
		if describeErr != nil {
			return describeErr
		}

		var err error
		if clusterType == Single {
			err = cache.nukeNonReplicationGroupElasticacheCluster(clusterId)
		} else if clusterType == Replication {
			err = cache.nukeReplicationGroupMemberElasticacheCluster(clusterId)
		}

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(clusterId),
			ResourceType: "Elasticache",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedClusterIds = append(deletedClusterIds, clusterId)
			logging.Debugf("Deleted Elasticache cluster: %s", *clusterId)
		}
	}

	logging.Debugf("[OK] %d Elasticache clusters deleted in %s", len(deletedClusterIds), cache.Region)
	return nil
}

// Custom errors

type CouldNotLookupCacheClusterErr struct {
	ClusterId *string
}

func (err CouldNotLookupCacheClusterErr) Error() string {
	return fmt.Sprintf("Failed to lookup clusterId: %s", aws.ToString(err.ClusterId))
}

/*
Elasticache Parameter Groups
*/

func (pg *ElasticacheParameterGroups) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var paramGroupNames []*string

	paginator := elasticache.NewDescribeCacheParameterGroupsPaginator(pg.Client, &elasticache.DescribeCacheParameterGroupsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, goerrors.WithStackTrace(err)
		}

		for _, paramGroup := range page.CacheParameterGroups {
			if pg.shouldInclude(&paramGroup, configObj) {
				paramGroupNames = append(paramGroupNames, paramGroup.CacheParameterGroupName)
			}
		}
	}

	return paramGroupNames, nil
}

func (pg *ElasticacheParameterGroups) shouldInclude(paramGroup *types.CacheParameterGroup, configObj config.Config) bool {
	if paramGroup == nil {
		return false
	}
	// Exclude AWS managed resources. user defined resources are unable to begin with "default."
	if strings.HasPrefix(aws.ToString(paramGroup.CacheParameterGroupName), "default.") {
		return false
	}

	return configObj.ElasticacheParameterGroups.ShouldInclude(config.ResourceValue{
		Name: paramGroup.CacheParameterGroupName,
	})
}

func (pg *ElasticacheParameterGroups) nukeAll(paramGroupNames []*string) error {
	if len(paramGroupNames) == 0 {
		logging.Debugf("No Elasticache parameter groups to nuke in region %s", pg.Region)
		return nil
	}
	var deletedGroupNames []*string
	for _, pgName := range paramGroupNames {
		_, err := pg.Client.DeleteCacheParameterGroup(pg.Context, &elasticache.DeleteCacheParameterGroupInput{CacheParameterGroupName: pgName})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(pgName),
			ResourceType: "Elasticache Parameter Group",
			Error:        err,
		}
		report.Record(e)
		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedGroupNames = append(deletedGroupNames, pgName)
			logging.Debugf("Deleted Elasticache parameter group: %s", aws.ToString(pgName))
		}
	}
	logging.Debugf("[OK] %d Elasticache parameter groups deleted in %s", len(deletedGroupNames), pg.Region)
	return nil
}

/*
Elasticache Subnet Groups
*/
func (sg *ElasticacheSubnetGroups) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var subnetGroupNames []*string

	paginator := elasticache.NewDescribeCacheSubnetGroupsPaginator(sg.Client, &elasticache.DescribeCacheSubnetGroupsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, goerrors.WithStackTrace(err)
		}

		for _, subnetGroup := range page.CacheSubnetGroups {
			if !strings.Contains(*subnetGroup.CacheSubnetGroupName, "default") &&
				configObj.ElasticacheSubnetGroups.ShouldInclude(config.ResourceValue{
					Name: subnetGroup.CacheSubnetGroupName,
				}) {
				subnetGroupNames = append(subnetGroupNames, subnetGroup.CacheSubnetGroupName)
			}
		}
	}

	return subnetGroupNames, nil
}

func (sg *ElasticacheSubnetGroups) nukeAll(subnetGroupNames []*string) error {
	if len(subnetGroupNames) == 0 {
		logging.Debugf("No Elasticache subnet groups to nuke in region %s", sg.Region)
		return nil
	}
	var deletedGroupNames []*string
	for _, sgName := range subnetGroupNames {
		_, err := sg.Client.DeleteCacheSubnetGroup(sg.Context, &elasticache.DeleteCacheSubnetGroupInput{CacheSubnetGroupName: sgName})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(sgName),
			ResourceType: "Elasticache Subnet Group",
			Error:        err,
		}
		report.Record(e)
		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedGroupNames = append(deletedGroupNames, sgName)
			logging.Debugf("Deleted Elasticache subnet group: %s", aws.ToString(sgName))
		}
	}
	logging.Debugf("[OK] %d Elasticache subnet groups deleted in %s", len(deletedGroupNames), sg.Region)
	return nil
}
