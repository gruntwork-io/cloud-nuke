package resources

import (
	"context"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
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
	result, err := instance.Client.DescribeGlobalClustersWithContext(instance.Context, &rds.DescribeGlobalClustersInput{})
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
		_, err := instance.Client.DeleteGlobalClusterWithContext(instance.Context, &rds.DeleteGlobalClusterInput{
			GlobalClusterIdentifier: name,
		})

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

		default:
			deletedNames = append(deletedNames, name)
			logging.Debugf("Deleted RDS DB Global Cluster Membership: %s", awsgo.StringValue(name))
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
		_, err := instance.Client.DescribeGlobalClustersWithContext(instance.Context, &rds.DescribeGlobalClustersInput{
			GlobalClusterIdentifier: &name,
		})
		if err != nil {
			if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == rds.ErrCodeGlobalClusterNotFoundFault {
				return nil
			}

			return errors.WithStackTrace(err)
		}

		time.Sleep(dbGlobalClusterDeletionRetryDelay)
		logging.Debug("Waiting for RDS Global Cluster to be deleted")
	}

	return RdsDeleteError{name: name}
}
