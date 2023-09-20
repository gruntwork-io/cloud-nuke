package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

func (rc *RedshiftClusters) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var clusterIds []*string
	err := rc.Client.DescribeClustersPages(
		&redshift.DescribeClustersInput{},
		func(page *redshift.DescribeClustersOutput, lastPage bool) bool {
			for _, cluster := range page.Clusters {
				if configObj.Redshift.ShouldInclude(config.ResourceValue{
					Time: cluster.ClusterCreateTime,
					Name: cluster.ClusterIdentifier,
				}) {
					clusterIds = append(clusterIds, cluster.ClusterIdentifier)
				}
			}

			return !lastPage
		},
	)

	return clusterIds, errors.WithStackTrace(err)
}

func (rc *RedshiftClusters) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Logger.Debugf("No Redshift Clusters to nuke in region %s", rc.Region)
		return nil
	}
	logging.Logger.Debugf("Deleting all Redshift Clusters in region %s", rc.Region)
	deletedIds := []*string{}
	for _, id := range identifiers {
		_, err := rc.Client.DeleteCluster(&redshift.DeleteClusterInput{
			ClusterIdentifier:        id,
			SkipFinalClusterSnapshot: aws.Bool(true),
		})
		if err != nil {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking RedshiftCluster",
			}, map[string]interface{}{
				"region": rc.Region,
			})
			logging.Logger.Errorf("[Failed] %s: %s", *id, err)
		} else {
			deletedIds = append(deletedIds, id)
			logging.Logger.Debugf("Deleted Redshift Cluster: %s", aws.StringValue(id))
		}
	}

	if len(deletedIds) > 0 {
		for _, id := range deletedIds {
			err := rc.Client.WaitUntilClusterDeleted(&redshift.DescribeClustersInput{ClusterIdentifier: id})
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
					"region": rc.Region,
				})
				logging.Logger.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}
	logging.Logger.Debugf("[OK] %d Redshift Cluster(s) deleted in %s", len(deletedIds), rc.Region)
	return nil
}
