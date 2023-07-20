package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"time"
)

func getAllRedshiftClusters(session *session.Session, region string, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := redshift.New(session)
	var clusterIds []*string
	err := svc.DescribeClustersPages(
		&redshift.DescribeClustersInput{},
		func(page *redshift.DescribeClustersOutput, lastPage bool) bool {
			for _, cluster := range page.Clusters {
				if shouldIncludeRedshiftCluster(cluster, excludeAfter, configObj) {
					clusterIds = append(clusterIds, cluster.ClusterIdentifier)
				}
			}
			return !lastPage
		},
	)
	return clusterIds, errors.WithStackTrace(err)
}

func shouldIncludeRedshiftCluster(cluster *redshift.Cluster, excludeAfter time.Time, configObj config.Config) bool {
	if cluster == nil {
		return false
	}
	if excludeAfter.Before(*cluster.ClusterCreateTime) {
		return false
	}
	return config.ShouldInclude(
		aws.StringValue(cluster.ClusterIdentifier),
		configObj.Redshift.IncludeRule.NamesRegExp,
		configObj.Redshift.ExcludeRule.NamesRegExp,
	)
}

func nukeAllRedshiftClusters(session *session.Session, identifiers []*string) error {
	svc := redshift.New(session)
	if len(identifiers) == 0 {
		logging.Logger.Debugf("No Redshift Clusters to nuke in region %s", *session.Config.Region)
		return nil
	}
	logging.Logger.Debugf("Deleting all Redshift Clusters in region %s", *session.Config.Region)
	deletedIds := []*string{}
	for _, id := range identifiers {
		_, err := svc.DeleteCluster(&redshift.DeleteClusterInput{ClusterIdentifier: id, SkipFinalClusterSnapshot: aws.Bool(true)})
		if err != nil {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking RedshiftClusters",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
			logging.Logger.Errorf("[Failed] %s: %s", *id, err)
		} else {
			deletedIds = append(deletedIds, id)
			logging.Logger.Debugf("Deleted Redshift Cluster: %s", aws.StringValue(id))
		}
	}

	if len(deletedIds) > 0 {
		for _, id := range deletedIds {
			err := svc.WaitUntilClusterDeleted(&redshift.DescribeClustersInput{ClusterIdentifier: id})
			// Record status of this resource
			e := report.Entry{
				Identifier:   aws.StringValue(id),
				ResourceType: "Redshift Cluster",
				Error:        err,
			}
			report.Record(e)
			if err != nil {
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: "Error Nuking Redshift Cluster",
				}, map[string]interface{}{
					"region": *session.Config.Region,
				})
				logging.Logger.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}
	logging.Logger.Debugf("[OK] %d Redshift Cluster(s) deleted in %s", len(deletedIds), *session.Config.Region)
	return nil
}
