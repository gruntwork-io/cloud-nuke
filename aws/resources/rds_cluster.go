package resources

import (
	"context"
	"time"

	"github.com/andrewderr/cloud-nuke-a1/util"

	awsgo "github.com/aws/aws-sdk-go/aws"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/andrewderr/cloud-nuke-a1/config"
	"github.com/andrewderr/cloud-nuke-a1/logging"
	"github.com/andrewderr/cloud-nuke-a1/report"
	"github.com/andrewderr/cloud-nuke-a1/telemetry"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gruntwork-io/go-commons/errors"
)

func (instance *DBClusters) waitUntilRdsClusterDeleted(input *rds.DescribeDBClustersInput) error {
	// wait up to 15 minutes
	for i := 0; i < 90; i++ {
		_, err := instance.Client.DescribeDBClusters(input)
		if err != nil {
			if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == rds.ErrCodeDBClusterNotFoundFault {
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
	result, err := instance.Client.DescribeDBClusters(&rds.DescribeDBClustersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string
	for _, database := range result.DBClusters {
		if configObj.DBClusters.ShouldInclude(config.ResourceValue{
			Name: database.DBClusterIdentifier,
			Time: database.ClusterCreateTime,
			Tags: util.ConvertRDSTagsToMap(database.TagList),
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
		params := &rds.DeleteDBClusterInput{
			DBClusterIdentifier: name,
			SkipFinalSnapshot:   awsgo.Bool(true),
		}

		_, err := instance.Client.DeleteDBCluster(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(name),
			ResourceType: "RDS Cluster",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s: %s", *name, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking RDS Cluster",
			}, map[string]interface{}{
				"region": instance.Region,
			})
		} else {
			deletedNames = append(deletedNames, name)
			logging.Debugf("Deleted RDS DB Cluster: %s", awsgo.StringValue(name))
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
