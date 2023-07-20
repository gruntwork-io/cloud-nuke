package aws

import (
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"strings"
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
	logging.Logger.Debugf("Deleting Elasticache cluster Id: %s which is not a member of a replication group", aws.StringValue(clusterId))
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
	logging.Logger.Debugf("Elasticache cluster Id: %s is a member of a replication group. Therefore, deleting its replication group", aws.StringValue(clusterId))

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

	logging.Logger.Debugf("Successfully deleted replication group Id: %s", aws.StringValue(clusterId))

	return nil
}

func nukeAllElasticacheClusters(session *session.Session, clusterIds []*string) error {
	svc := elasticache.New(session)

	if len(clusterIds) == 0 {
		logging.Logger.Debugf("No Elasticache clusters to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting %d Elasticache clusters in region %s", len(clusterIds), *session.Config.Region)

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
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Elasticache Cluster",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
		} else {
			deletedClusterIds = append(deletedClusterIds, clusterId)
			logging.Logger.Debugf("Deleted Elasticache cluster: %s", *clusterId)
		}
	}

	logging.Logger.Debugf("[OK] %d Elasticache clusters deleted in %s", len(deletedClusterIds), *session.Config.Region)
	return nil
}

// Custom errors

type CouldNotLookupCacheClusterErr struct {
	ClusterId *string
}

func (err CouldNotLookupCacheClusterErr) Error() string {
	return fmt.Sprintf("Failed to lookup clusterId: %s", aws.StringValue(err.ClusterId))
}

/*
Elasticache Parameter Groups
*/

func getAllElasticacheParameterGroups(session *session.Session, region string, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := elasticache.New(session)
	var paramGroupNames []*string
	err := svc.DescribeCacheParameterGroupsPages(
		&elasticache.DescribeCacheParameterGroupsInput{},
		func(page *elasticache.DescribeCacheParameterGroupsOutput, lastPage bool) bool {
			for _, paramGroup := range page.CacheParameterGroups {
				if shouldIncludeElasticacheParameterGroup(paramGroup, configObj) {
					paramGroupNames = append(paramGroupNames, paramGroup.CacheParameterGroupName)
				}
			}
			return !lastPage
		},
	)

	return paramGroupNames, errors.WithStackTrace(err)
}

func shouldIncludeElasticacheParameterGroup(paramGroup *elasticache.CacheParameterGroup, configObj config.Config) bool {
	if paramGroup == nil {
		return false
	}
	//Exclude AWS managed resources. user defined resources are unable to begin with "default."
	if strings.HasPrefix(aws.StringValue(paramGroup.CacheParameterGroupName), "default.") {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(paramGroup.CacheParameterGroupName),
		configObj.ElasticacheParameterGroup.IncludeRule.NamesRegExp,
		configObj.ElasticacheParameterGroup.ExcludeRule.NamesRegExp,
	)
}

func nukeAllElasticacheParameterGroups(session *session.Session, paramGroupNames []*string) error {
	svc := elasticache.New(session)
	if len(paramGroupNames) == 0 {
		logging.Logger.Debugf("No Elasticache parameter groups to nuke in region %s", *session.Config.Region)
		return nil
	}
	var deletedGroupNames []*string
	for _, pgName := range paramGroupNames {
		_, err := svc.DeleteCacheParameterGroup(&elasticache.DeleteCacheParameterGroupInput{CacheParameterGroupName: pgName})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(pgName),
			ResourceType: "Elasticache Parameter Group",
			Error:        err,
		}
		report.Record(e)
		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Elasticache Parameter Group",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
		} else {
			deletedGroupNames = append(deletedGroupNames, pgName)
			logging.Logger.Debugf("Deleted Elasticache parameter group: %s", aws.StringValue(pgName))
		}
	}
	logging.Logger.Debugf("[OK] %d Elasticache parameter groups deleted in %s", len(deletedGroupNames), *session.Config.Region)
	return nil
}

/*
Elasticache Subnet Groups
*/
func getAllElasticacheSubnetGroups(session *session.Session, region string, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := elasticache.New(session)
	var subnetGroupNames []*string
	err := svc.DescribeCacheSubnetGroupsPages(
		&elasticache.DescribeCacheSubnetGroupsInput{},
		func(page *elasticache.DescribeCacheSubnetGroupsOutput, lastPage bool) bool {
			for _, subnetGroup := range page.CacheSubnetGroups {
				if shouldIncludeElasticacheSubnetGroup(subnetGroup, configObj) {
					subnetGroupNames = append(subnetGroupNames, subnetGroup.CacheSubnetGroupName)
				}
			}
			return !lastPage
		},
	)

	return subnetGroupNames, errors.WithStackTrace(err)
}

func shouldIncludeElasticacheSubnetGroup(subnetGroup *elasticache.CacheSubnetGroup, configObj config.Config) bool {
	if subnetGroup == nil {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(subnetGroup.CacheSubnetGroupName),
		configObj.ElasticacheSubnetGroup.IncludeRule.NamesRegExp,
		configObj.ElasticacheSubnetGroup.ExcludeRule.NamesRegExp,
	)
}

func nukeAllElasticacheSubnetGroups(session *session.Session, subnetGroupNames []*string) error {
	svc := elasticache.New(session)
	if len(subnetGroupNames) == 0 {
		logging.Logger.Debugf("No Elasticache subnet groups to nuke in region %s", *session.Config.Region)
		return nil
	}
	var deletedGroupNames []*string
	for _, sgName := range subnetGroupNames {
		_, err := svc.DeleteCacheSubnetGroup(&elasticache.DeleteCacheSubnetGroupInput{CacheSubnetGroupName: sgName})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(sgName),
			ResourceType: "Elasticache Subnet Group",
			Error:        err,
		}
		report.Record(e)
		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Elasticache Subnet Group",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
		} else {
			deletedGroupNames = append(deletedGroupNames, sgName)
			logging.Logger.Debugf("Deleted Elasticache subnet group: %s", aws.StringValue(sgName))
		}
	}
	logging.Logger.Debugf("[OK] %d Elasticache subnet groups deleted in %s", len(deletedGroupNames), *session.Config.Region)
	return nil
}
