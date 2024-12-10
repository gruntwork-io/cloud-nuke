package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (rc *RedshiftClusters) getAll(ctx context.Context, configObj config.Config) ([]*string, error) {
	var clusterIds []*string

	// Initialize the paginator with any optional settings.
	paginator := redshift.NewDescribeClustersPaginator(rc.Client, &redshift.DescribeClustersInput{})

	// Use the paginator to go through each page of clusters.
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[Redshift] Failed to list clusters: %s", err)
			return nil, errors.WithStackTrace(err)
		}

		// Process each cluster in the current page.
		for _, cluster := range output.Clusters {
			if configObj.Redshift.ShouldInclude(config.ResourceValue{
				Time: cluster.ClusterCreateTime,
				Name: cluster.ClusterIdentifier,
			}) {
				clusterIds = append(clusterIds, cluster.ClusterIdentifier)
			}
		}
	}

	return clusterIds, nil
}

func (rc *RedshiftClusters) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Redshift Clusters to nuke in region %s", rc.Region)
		return nil
	}
	logging.Debugf("Deleting all Redshift Clusters in region %s", rc.Region)
	deletedIds := []*string{}
	for _, id := range identifiers {
		_, err := rc.Client.DeleteCluster(rc.Context, &redshift.DeleteClusterInput{
			ClusterIdentifier:        id,
			SkipFinalClusterSnapshot: aws.Bool(true),
		})
		if err != nil {
			logging.Errorf("[Failed] %s: %s", *id, err)
		} else {
			deletedIds = append(deletedIds, id)
			logging.Debugf("Deleted Redshift Cluster: %s", aws.ToString(id))
		}
	}

	if len(deletedIds) > 0 {
		for _, id := range deletedIds {
			waiter := redshift.NewClusterDeletedWaiter(rc.Client)
			err := waiter.Wait(rc.Context, &redshift.DescribeClustersInput{
				ClusterIdentifier: id,
			}, 5*time.Minute)

			// Record status of this resource
			e := report.Entry{
				Identifier:   aws.ToString(id),
				ResourceType: "Redshift Cluster",
				Error:        err,
			}
			report.Record(e)
			if err != nil {
				logging.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}
	logging.Debugf("[OK] %d Redshift Cluster(s) deleted in %s", len(deletedIds), rc.Region)
	return nil
}
