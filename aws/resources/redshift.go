package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
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
		Nuker:  resource.SequentialDeleter(resource.DeleteThenWait(deleteRedshiftCluster, waitForRedshiftClusterDeleted)),
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

// deleteRedshiftCluster deletes a single Redshift cluster.
func deleteRedshiftCluster(ctx context.Context, client RedshiftClustersAPI, id *string) error {
	_, err := client.DeleteCluster(ctx, &redshift.DeleteClusterInput{
		ClusterIdentifier:        id,
		SkipFinalClusterSnapshot: aws.Bool(true),
	})
	return err
}

// waitForRedshiftClusterDeleted waits for a Redshift cluster to be deleted.
func waitForRedshiftClusterDeleted(ctx context.Context, client RedshiftClustersAPI, id *string) error {
	waiter := redshift.NewClusterDeletedWaiter(client)
	return waiter.Wait(ctx, &redshift.DescribeClustersInput{
		ClusterIdentifier: id,
	}, 5*time.Minute)
}
