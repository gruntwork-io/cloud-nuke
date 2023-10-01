package resources

import (
	"context"
	"fmt"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/pterm/pterm"
	"strings"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
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
		pterm.Debug.Println(fmt.Sprintf("Waiting for RDS Cluster %s to be deleted", *input.DBClusterIdentifier))
	}

	return RdsDeleteError{name: *input.DBClusterIdentifier}
}

func (instance *DBClusters) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := instance.Client.DescribeDBClusters(&rds.DescribeDBClustersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var identifiers []*string
	var identifierToArnMap = make(map[string]*string)
	for _, database := range result.DBClusters {
		if configObj.DBClusters.ShouldInclude(config.ResourceValue{
			Name: database.DBClusterIdentifier,
			Time: database.ClusterCreateTime,
			Tags: util.ConvertRDSTagsToMap(database.TagList),
		}) {
			identifiers = append(identifiers, database.DBClusterIdentifier)
			identifierToArnMap[*database.DBClusterIdentifier] = database.DBClusterArn
		}
	}

	instance.IdentifierToArnMap = identifierToArnMap
	return identifiers, nil
}

func (instance *DBClusters) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Logger.Debugf("No RDS DB Cluster to nuke in region %s", instance.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all RDS Clusters in region %s", instance.Region)
	deletedNames := []*string{}

	// TODO: potentially add support for pagination in case there are too many global clusters.
	globalClusters, err := instance.Client.DescribeGlobalClusters(&rds.DescribeGlobalClustersInput{})
	if err != nil {
		pterm.Debug.Println(fmt.Sprintf("Failed to describe global clusters: %s", err))
		return errors.WithStackTrace(err)
	}

	pterm.Debug.Println(fmt.Sprintf("Found %d global clusters", len(globalClusters.GlobalClusters)))
	for _, globalCluster := range globalClusters.GlobalClusters {
		for _, member := range globalCluster.GlobalClusterMembers {
			for _, identifier := range identifiers {
				if strings.Contains(*member.DBClusterArn, *identifier) {
					dbClusterArn := instance.IdentifierToArnMap[*identifier]
					_, err := instance.Client.RemoveFromGlobalCluster(&rds.RemoveFromGlobalClusterInput{
						GlobalClusterIdentifier: globalCluster.GlobalClusterIdentifier,
						DbClusterIdentifier:     dbClusterArn,
					})
					if err != nil {
						pterm.Debug.Println(fmt.Sprintf(
							"Failed to remove cluster %s from global cluster %s: %s",
							*identifier, *globalCluster.GlobalClusterIdentifier, err))
						return err
					}

					pterm.Debug.Println(fmt.Sprintf(
						"Successfully removed cluster %s from global cluster %s",
						*identifier, *globalCluster.GlobalClusterIdentifier))
				}
			}
		}
	}

	for _, name := range identifiers {
		pterm.Debug.Println("Deleting RDS Cluster: ", *name)
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
			pterm.Debug.Println("Failed to delete RDS Cluster: ", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking RDS Cluster",
			}, map[string]interface{}{
				"region": instance.Region,
			})
		} else {
			pterm.Debug.Println("Successfully deleted RDS Cluster: ", *name)
			deletedNames = append(deletedNames, name)
		}
	}

	if len(deletedNames) > 0 {
		for _, name := range deletedNames {

			err := instance.waitUntilRdsClusterDeleted(&rds.DescribeDBClustersInput{
				DBClusterIdentifier: name,
			})
			if err != nil {
				pterm.Debug.Println(fmt.Sprintf("Failed to wait for RDS Cluster %s to be deleted: %s", *name, err))
				return errors.WithStackTrace(err)
			}
		}
	}

	pterm.Debug.Println(fmt.Sprintf("Deleted %d RDS Clusters in %s", len(deletedNames), instance.Region))
	return nil
}
