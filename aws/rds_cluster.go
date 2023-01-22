package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func waitUntilRdsClusterDeleted(svc *rds.RDS, input *rds.DescribeDBClustersInput) error {
	// wait up to 15 minutes
	for i := 0; i < 90; i++ {
		_, err := svc.DescribeDBClusters(input)
		if err != nil {
			if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == rds.ErrCodeDBClusterNotFoundFault {
				return nil
			}

			return err
		}

		time.Sleep(10 * time.Second)
		logging.Logger.Debug("Waiting for RDS Cluster to be deleted")
	}

	return RdsDeleteError{name: *input.DBClusterIdentifier}
}

func getAllRdsClusters(session *session.Session, excludeAfter time.Time) ([]*string, error) {
	svc := rds.New(session)

	result, err := svc.DescribeDBClusters(&rds.DescribeDBClustersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string

	for _, database := range result.DBClusters {
		if excludeAfter.After(*database.ClusterCreateTime) && !hasRDSClusterExcludeTag(database) {
			names = append(names, database.DBClusterIdentifier)
		} else if !hasRDSClusterExcludeTag(database) {
			names = append(names, database.DBClusterIdentifier)
		}
	}

	return names, nil
}

// hasRDSClusterExcludeTag checks whether the exlude tag is set for a resource to skip deleting it.
func hasRDSClusterExcludeTag(database *rds.DBCluster) bool {
	// Exclude deletion of any RDS cluster (Aurora) with cloud-nuke-excluded tags
	for _, tag := range database.TagList {
		if *tag.Key == AwsResourceExclusionTagKey && *tag.Value == "true" {
			return true
		}
	}
	return false
}

func nukeAllRdsClusters(session *session.Session, names []*string) error {
	svc := rds.New(session)

	if len(names) == 0 {
		logging.Logger.Debugf("No RDS DB Cluster to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all RDS Clusters in region %s", *session.Config.Region)
	deletedNames := []*string{}

	for _, name := range names {
		params := &rds.DeleteDBClusterInput{
			DBClusterIdentifier: name,
			SkipFinalSnapshot:   awsgo.Bool(true),
		}

		_, err := svc.DeleteDBCluster(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(name),
			ResourceType: "RDS Cluster",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("[Failed] %s: %s", *name, err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Logger.Debugf("Deleted RDS DB Cluster: %s", awsgo.StringValue(name))
		}
	}

	if len(deletedNames) > 0 {
		for _, name := range deletedNames {

			err := waitUntilRdsClusterDeleted(svc, &rds.DescribeDBClustersInput{
				DBClusterIdentifier: name,
			})
			if err != nil {
				logging.Logger.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}

	logging.Logger.Debugf("[OK] %d RDS DB Cluster(s) nuked in %s", len(deletedNames), *session.Config.Region)
	return nil
}
