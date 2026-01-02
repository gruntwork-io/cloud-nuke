package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// DBClustersAPI defines the interface for RDS DB Cluster operations.
type DBClustersAPI interface {
	DeleteDBCluster(ctx context.Context, params *rds.DeleteDBClusterInput, optFns ...func(*rds.Options)) (*rds.DeleteDBClusterOutput, error)
	DescribeDBClusters(ctx context.Context, params *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error)
	ModifyDBCluster(ctx context.Context, params *rds.ModifyDBClusterInput, optFns ...func(*rds.Options)) (*rds.ModifyDBClusterOutput, error)
}

// NewDBClusters creates a new RDS DB Clusters resource using the generic resource pattern.
func NewDBClusters() AwsResource {
	return NewAwsResource(&resource.Resource[DBClustersAPI]{
		ResourceTypeName: "rds-cluster",
		BatchSize:        49,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[DBClustersAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = rds.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.DBClusters.ResourceType
		},
		Lister: listDBClusters,
		Nuker:  resource.SequentialDeleteThenWaitAll(deleteDBCluster, waitForDBClustersDeleted),
	})
}

// listDBClusters retrieves all RDS DB Clusters that match the config filters.
func listDBClusters(ctx context.Context, client DBClustersAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var names []*string
	paginator := rds.NewDescribeDBClustersPaginator(client, &rds.DescribeDBClustersInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, cluster := range page.DBClusters {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: cluster.DBClusterIdentifier,
				Time: cluster.ClusterCreateTime,
				Tags: util.ConvertRDSTypeTagsToMap(cluster.TagList),
			}) {
				names = append(names, cluster.DBClusterIdentifier)
			}
		}
	}

	return names, nil
}

// deleteDBCluster deletes a single RDS DB Cluster after disabling deletion protection.
func deleteDBCluster(ctx context.Context, client DBClustersAPI, name *string) error {
	if _, err := client.ModifyDBCluster(ctx, &rds.ModifyDBClusterInput{
		DBClusterIdentifier: name,
		DeletionProtection:  aws.Bool(false),
		ApplyImmediately:    aws.Bool(true),
	}); err != nil {
		logging.Warnf("[Failed] to disable deletion protection for cluster %s: %s", aws.ToString(name), err)
	}

	_, err := client.DeleteDBCluster(ctx, &rds.DeleteDBClusterInput{
		DBClusterIdentifier: name,
		SkipFinalSnapshot:   aws.Bool(true),
	})
	return errors.WithStackTrace(err)
}

// waitForDBClustersDeleted waits for all specified DB clusters to be deleted.
func waitForDBClustersDeleted(ctx context.Context, client DBClustersAPI, ids []string) error {
	waiter := rds.NewDBClusterDeletedWaiter(client)
	for _, id := range ids {
		if err := waiter.Wait(ctx, &rds.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(id),
		}, 5*time.Minute); err != nil {
			return err
		}
	}
	return nil
}
