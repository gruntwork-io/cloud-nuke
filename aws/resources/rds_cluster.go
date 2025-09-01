package resources

import (
	"context"
	goerr "errors"
	"time"

	"github.com/gruntwork-io/cloud-nuke/util"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (instance *DBClusters) waitUntilRdsClusterDeleted(input *rds.DescribeDBClustersInput) error {
	// wait up to 15 minutes
	for i := 0; i < 90; i++ {
		_, err := instance.Client.DescribeDBClusters(instance.Context, input)
		if err != nil {
			var notFoundErr *types.DBClusterNotFoundFault
			if goerr.As(err, &notFoundErr) {
				return nil
			}

			return err
		}

		time.Sleep(10 * time.Second)
		logging.Debug("Waiting for RDS Cluster to be deleted")
	}

	return RdsDeleteError{name: *input.DBClusterIdentifier}
}

func (instance *DBClusters) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := instance.Client.DescribeDBClusters(c, &rds.DescribeDBClustersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string
	for _, database := range result.DBClusters {
		// Skip deletion-protected clusters while config object doesn't include deletion-protected
		if database.DeletionProtection != nil && *database.DeletionProtection && !configObj.DBClusters.IncludeDeletionProtected {
			continue
		}

		if configObj.DBClusters.ShouldInclude(config.ResourceValue{
			Name: database.DBClusterIdentifier,
			Time: database.ClusterCreateTime,
			Tags: util.ConvertRDSTypeTagsToMap(database.TagList),
		}) {
			names = append(names, database.DBClusterIdentifier)
		}
	}

	return names, nil
}

func (instance *DBClusters) nukeAll(names []*string) error {
	if len(names) == 0 {
		logging.Debugf("No RDS DB Cluster to nuke in region %s", instance.Region)
		return nil
	}

	logging.Debugf("Deleting all RDS Clusters in region %s", instance.Region)
	deletedNames := []*string{}

	for _, name := range names {
		// Disable deletion protection
		_, err := instance.Client.ModifyDBCluster(instance.Context, &rds.ModifyDBClusterInput{
			DBClusterIdentifier: name,
			DeletionProtection:  aws.Bool(false),
			ApplyImmediately:    aws.Bool(true),
		})
		if err != nil {
			logging.Warnf("[Failed] to disable deletion protection for cluster %s: %s", *name, err)
		}

		params := &rds.DeleteDBClusterInput{
			DBClusterIdentifier: name,
			SkipFinalSnapshot:   aws.Bool(true),
		}

		_, err = instance.Client.DeleteDBCluster(instance.Context, params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(name),
			ResourceType: "RDS Cluster",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s: %s", *name, err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Debugf("Deleted RDS DB Cluster: %s", aws.ToString(name))
		}
	}

	if len(deletedNames) > 0 {
		for _, name := range deletedNames {

			err := instance.waitUntilRdsClusterDeleted(&rds.DescribeDBClustersInput{
				DBClusterIdentifier: name,
			})
			if err != nil {
				logging.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}

	logging.Debugf("[OK] %d RDS DB Cluster(s) nuked in %s", len(deletedNames), instance.Region)
	return nil
}
