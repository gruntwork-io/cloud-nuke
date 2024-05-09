package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// wait up to 15 minutes
const (
	dbGlobalClusterMembershipsRemovalRetryDelay = 10 * time.Second
	dbGlobalClusterMembershipsRemovalRetryCount = 90
)

func (instance *DBGlobalClusterMemberships) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := instance.Client.DescribeGlobalClustersWithContext(instance.Context, &rds.DescribeGlobalClustersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string
	for _, cluster := range result.GlobalClusters {
		if !configObj.DBGlobalClusterMemberships.ShouldInclude(config.ResourceValue{
			Name: cluster.GlobalClusterIdentifier,
		}) {
			continue
		}

		names = append(names, cluster.GlobalClusterIdentifier)
	}

	return names, nil
}

func (instance *DBGlobalClusterMemberships) nukeAll(names []*string) error {
	if len(names) == 0 {
		logging.Debugf("No RDS DB Global Cluster Membership to nuke")
		return nil
	}

	logging.Debugf("Deleting Global Cluster (members)")
	deletedNames := []*string{}

	for _, name := range names {
		deleted, err := instance.removeGlobalClusterMembership(*name)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(name),
			ResourceType: "RDS Global Cluster Membership",
			Error:        err,
		}
		report.Record(e)

		switch {
		case err != nil:
			logging.Debugf("[Failed] %s: %s", *name, err)

		case !deleted:
			logging.Debugf("No RDS Global Cluster Membership was deleted on %s", *name)

		default:
			deletedNames = append(deletedNames, name)
			logging.Debugf("Deleted RDS DB Global Cluster Membership: %s", awsgo.StringValue(name))
		}
	}

	logging.Debugf("[OK] %d RDS Global DB Cluster(s) Membership nuked in %s", len(deletedNames), instance.Region)
	return nil
}

func (instance *DBGlobalClusterMemberships) removeGlobalClusterMembership(name string) (deleted bool, err error) {
	gdbcs, err := instance.Client.DescribeGlobalClustersWithContext(instance.Context, &rds.DescribeGlobalClustersInput{
		GlobalClusterIdentifier: &name,
	})
	if err != nil {
		return deleted, fmt.Errorf("fail to describe global cluster: %w", err)
	}
	if len(gdbcs.GlobalClusters) != 1 || *gdbcs.GlobalClusters[0].GlobalClusterIdentifier != name {
		return deleted, fmt.Errorf("unexpected describe result global cluster")
	}
	gdbc := gdbcs.GlobalClusters[0]

	deletedNames := []string{}
	for _, member := range gdbc.GlobalClusterMembers {
		region := strings.Split(*member.DBClusterArn, ":")[3]
		if instance.Region != "" && instance.Region != region {
			logging.Debugf("Skip removing cluster '%s' from global cluster since it is in different region", *member.DBClusterArn)
			continue
		}

		logging.Debugf("Removing cluster '%s' from global cluster", *member.DBClusterArn)
		_, err := instance.Client.RemoveFromGlobalClusterWithContext(instance.Context, &rds.RemoveFromGlobalClusterInput{
			GlobalClusterIdentifier: gdbc.GlobalClusterIdentifier,
			DbClusterIdentifier:     member.DBClusterArn,
		})
		if err != nil {
			return deleted, fmt.Errorf("fail to remove cluster '%s' from global cluster :%w", *member, err)
		}
		deletedNames = append(deletedNames, *member.DBClusterArn)
	}
	for _, name := range deletedNames {
		err = instance.waitUntilRdsClusterRemovedFromGlobalCluster(*gdbc.GlobalClusterIdentifier, name)
		if err != nil {
			return deleted, fmt.Errorf("fail to remove cluster '%s' from global cluster :%w", name, err)
		}
	}

	return len(deletedNames) > 0, nil
}

func (instance *DBGlobalClusterMemberships) waitUntilRdsClusterRemovedFromGlobalCluster(arnGlobalCluster string, arnCluster string) error {
	for i := 0; i < dbGlobalClusterMembershipsRemovalRetryCount; i++ {
		gcs, err := instance.Client.DescribeGlobalClustersWithContext(instance.Context, &rds.DescribeGlobalClustersInput{
			GlobalClusterIdentifier: &arnGlobalCluster,
		})
		if err != nil {
			return errors.WithStackTrace(err)
		}

		found := false
		for _, gc := range gcs.GlobalClusters {
			for _, m := range gc.GlobalClusterMembers {
				if *m.DBClusterArn != arnCluster {
					continue
				}

				found = true
				break
			}
		}
		if !found {
			return nil
		}

		time.Sleep(dbGlobalClusterMembershipsRemovalRetryDelay)
		logging.Debug("Waiting for RDS Cluster to be removed from RDS Global Cluster")
	}

	return RdsDeleteError{name: arnCluster}
}
