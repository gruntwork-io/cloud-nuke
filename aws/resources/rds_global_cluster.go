package resources

import (
	"context"
	goerr "errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// wait up to 15 minutes
const (
	dbGlobalClusterDeletionRetryDelay = 10 * time.Second
	dbGlobalClusterDeletionRetryCount = 90
)

func (instance *DBGlobalClusters) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := instance.Client.DescribeGlobalClusters(instance.Context, &rds.DescribeGlobalClustersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string
	for _, cluster := range result.GlobalClusters {
		if !configObj.DBGlobalClusters.ShouldInclude(config.ResourceValue{
			Name: cluster.GlobalClusterIdentifier,
		}) {
			continue
		}

		names = append(names, cluster.GlobalClusterIdentifier)
	}

	return names, nil
}

func (instance *DBGlobalClusters) nukeAll(names []*string) error {
	if len(names) == 0 {
		logging.Debugf("No RDS DB Global Cluster Membership to nuke")
		return nil
	}

	logging.Debugf("Deleting Global Cluster (members)")
	deletedNames := []*string{}

	for _, name := range names {
		_, err := instance.Client.DeleteGlobalCluster(instance.Context, &rds.DeleteGlobalClusterInput{
			GlobalClusterIdentifier: name,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(name),
			ResourceType: "RDS Global Cluster Membership",
			Error:        err,
		}
		report.Record(e)

		switch {
		case err != nil:
			logging.Debugf("[Failed] %s: %s", *name, err)

		default:
			deletedNames = append(deletedNames, name)
			logging.Debugf("Deleted RDS DB Global Cluster Membership: %s", aws.ToString(name))
		}
	}

	for _, name := range deletedNames {
		err := instance.waitUntilRDSGlobalClusterDeleted(*name)
		if err != nil {
			logging.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
	}

	logging.Debugf("[OK] %d RDS Global DB Cluster(s) Membership nuked in %s", len(deletedNames), instance.Region)
	return nil
}

func (instance *DBGlobalClusters) waitUntilRDSGlobalClusterDeleted(name string) error {
	for i := 0; i < dbGlobalClusterDeletionRetryCount; i++ {
		_, err := instance.Client.DescribeGlobalClusters(instance.Context, &rds.DescribeGlobalClustersInput{
			GlobalClusterIdentifier: &name,
		})
		if err != nil {
			var notFoundErr *types.GlobalClusterNotFoundFault
			if goerr.As(err, &notFoundErr) {
				return nil
			}

			return errors.WithStackTrace(err)
		}

		time.Sleep(dbGlobalClusterDeletionRetryDelay)
		logging.Debug("Waiting for RDS Global Cluster to be deleted")
	}

	return RdsDeleteError{name: name}
}
