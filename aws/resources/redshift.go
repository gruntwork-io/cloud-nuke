package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// RedshiftClustersAPI defines the interface for Redshift Cluster operations.
type RedshiftClustersAPI interface {
	DescribeClusters(ctx context.Context, params *redshift.DescribeClustersInput, optFns ...func(*redshift.Options)) (*redshift.DescribeClustersOutput, error)
	DeleteCluster(ctx context.Context, params *redshift.DeleteClusterInput, optFns ...func(*redshift.Options)) (*redshift.DeleteClusterOutput, error)
}

// NewRedshiftClusters creates a new RedshiftClusters resource using the generic resource pattern.
func NewRedshiftClusters() AwsResource {
	return NewAwsResource(&resource.Resource[RedshiftClustersAPI]{
		ResourceTypeName: "redshift",
		BatchSize:        49,
		InitClient: func(r *resource.Resource[RedshiftClustersAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for Redshift client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = redshift.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.Redshift
		},
		Lister: listRedshiftClusters,
		Nuker:  deleteRedshiftClusters,
	})
}

// listRedshiftClusters retrieves all Redshift clusters that match the config filters.
func listRedshiftClusters(ctx context.Context, client RedshiftClustersAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var clusterIds []*string

	paginator := redshift.NewDescribeClustersPaginator(client, &redshift.DescribeClustersInput{})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			logging.Debugf("[Redshift] Failed to list clusters: %s", err)
			return nil, errors.WithStackTrace(err)
		}

		for _, cluster := range output.Clusters {
			if cfg.ShouldInclude(config.ResourceValue{
				Time: cluster.ClusterCreateTime,
				Name: cluster.ClusterIdentifier,
			}) {
				clusterIds = append(clusterIds, cluster.ClusterIdentifier)
			}
		}
	}

	return clusterIds, nil
}

// deleteRedshiftClusters deletes all Redshift clusters.
func deleteRedshiftClusters(ctx context.Context, client RedshiftClustersAPI, scope resource.Scope, resourceType string, identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Redshift Clusters to nuke in region %s", scope.Region)
		return nil
	}

	logging.Debugf("Deleting all Redshift Clusters in region %s", scope.Region)
	deletedIds := []*string{}

	for _, id := range identifiers {
		_, err := client.DeleteCluster(ctx, &redshift.DeleteClusterInput{
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
			waiter := redshift.NewClusterDeletedWaiter(client)
			err := waiter.Wait(ctx, &redshift.DescribeClustersInput{
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

	logging.Debugf("[OK] %d Redshift Cluster(s) deleted in %s", len(deletedIds), scope.Region)
	return nil
}
