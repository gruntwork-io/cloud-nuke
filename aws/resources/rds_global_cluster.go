package resources

import (
	"context"
	goerr "errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// DBGlobalClustersAPI defines the interface for RDS Global Cluster operations.
type DBGlobalClustersAPI interface {
	DescribeGlobalClusters(ctx context.Context, params *rds.DescribeGlobalClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeGlobalClustersOutput, error)
	DeleteGlobalCluster(ctx context.Context, params *rds.DeleteGlobalClusterInput, optFns ...func(*rds.Options)) (*rds.DeleteGlobalClusterOutput, error)
}

// NewDBGlobalClusters creates a new RDS Global Clusters resource using the generic resource pattern.
func NewDBGlobalClusters() AwsResource {
	return NewAwsResource(&resource.Resource[DBGlobalClustersAPI]{
		ResourceTypeName: "rds-global-cluster",
		BatchSize:        DefaultBatchSize,
		IsGlobal:         true,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[DBGlobalClustersAPI], cfg aws.Config) {
			r.Scope.Region = "global"
			r.Client = rds.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.DBGlobalClusters
		},
		Lister: listDBGlobalClusters,
		Nuker:  resource.SequentialDeleteThenWaitAll(deleteDBGlobalCluster, waitForDBGlobalClustersDeleted),
	})
}

// listDBGlobalClusters retrieves all RDS Global Clusters that match the config filters.
func listDBGlobalClusters(ctx context.Context, client DBGlobalClustersAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var names []*string
	paginator := rds.NewDescribeGlobalClustersPaginator(client, &rds.DescribeGlobalClustersInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, cluster := range page.GlobalClusters {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: cluster.GlobalClusterIdentifier,
			}) {
				names = append(names, cluster.GlobalClusterIdentifier)
			}
		}
	}

	return names, nil
}

// deleteDBGlobalCluster deletes a single RDS Global Cluster.
func deleteDBGlobalCluster(ctx context.Context, client DBGlobalClustersAPI, name *string) error {
	_, err := client.DeleteGlobalCluster(ctx, &rds.DeleteGlobalClusterInput{
		GlobalClusterIdentifier: name,
	})
	return err
}

// waitForDBGlobalClustersDeleted waits for all specified global clusters to be deleted.
// AWS SDK doesn't provide a waiter for global cluster deletion, so we poll manually.
func waitForDBGlobalClustersDeleted(ctx context.Context, client DBGlobalClustersAPI, ids []string) error {
	const (
		retryDelay = 10 * time.Second
		maxRetries = 90 // up to 15 minutes
	)

	for _, name := range ids {
		for i := 0; i < maxRetries; i++ {
			_, err := client.DescribeGlobalClusters(ctx, &rds.DescribeGlobalClustersInput{
				GlobalClusterIdentifier: aws.String(name),
			})
			if err != nil {
				var notFoundErr *types.GlobalClusterNotFoundFault
				if goerr.As(err, &notFoundErr) {
					break // Cluster is deleted
				}
				return errors.WithStackTrace(err)
			}

			logging.Debugf("Waiting for RDS Global Cluster %s to be deleted", name)
			time.Sleep(retryDelay)
		}

		// Check one more time after max retries
		_, err := client.DescribeGlobalClusters(ctx, &rds.DescribeGlobalClustersInput{
			GlobalClusterIdentifier: aws.String(name),
		})
		if err == nil {
			return fmt.Errorf("RDS global cluster %s was not deleted within timeout", name)
		}
		var notFoundErr *types.GlobalClusterNotFoundFault
		if !goerr.As(err, &notFoundErr) {
			return errors.WithStackTrace(err)
		}
	}

	return nil
}
