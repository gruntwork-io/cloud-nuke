package resources

import (
	"context"
	goerr "errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
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
		InitClient: func(r *resource.Resource[DBClustersAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for RDS client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = rds.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.DBClusters.ResourceType
		},
		Lister: listDBClusters,
		Nuker:  resource.SequentialDeleter(resource.DeleteThenWait(deleteDBCluster, waitUntilRdsClusterDeleted)),
	})
}

// listDBClusters retrieves all RDS DB Clusters that match the config filters.
func listDBClusters(ctx context.Context, client DBClustersAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	result, err := client.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{})
	if err != nil {
		return nil, err
	}

	var names []*string
	for _, database := range result.DBClusters {
		if cfg.ShouldInclude(config.ResourceValue{
			Name: database.DBClusterIdentifier,
			Time: database.ClusterCreateTime,
			Tags: util.ConvertRDSTypeTagsToMap(database.TagList),
		}) {
			names = append(names, database.DBClusterIdentifier)
		}
	}

	return names, nil
}

// waitUntilRdsClusterDeleted waits until the RDS cluster is deleted.
func waitUntilRdsClusterDeleted(ctx context.Context, client DBClustersAPI, clusterIdentifier *string) error {
	waitTimeout := DefaultWaitTimeout
	const retryInterval = 10 * time.Second
	maxRetries := int(waitTimeout / retryInterval)

	for i := 0; i < maxRetries; i++ {
		_, err := client.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{
			DBClusterIdentifier: clusterIdentifier,
		})
		if err != nil {
			var notFoundErr *types.DBClusterNotFoundFault
			if goerr.As(err, &notFoundErr) {
				return nil
			}
			return err
		}

		time.Sleep(retryInterval)
		logging.Debug("Waiting for RDS Cluster to be deleted")
	}

	return RdsDeleteError{name: aws.ToString(clusterIdentifier)}
}

// deleteDBCluster deletes a single RDS DB Cluster after disabling deletion protection.
func deleteDBCluster(ctx context.Context, client DBClustersAPI, name *string) error {
	// Disable deletion protection before attempting to delete the cluster
	_, err := client.ModifyDBCluster(ctx, &rds.ModifyDBClusterInput{
		DBClusterIdentifier: name,
		DeletionProtection:  aws.Bool(false),
		ApplyImmediately:    aws.Bool(true),
	})
	if err != nil {
		logging.Warnf("[Failed] to disable deletion protection for cluster %s: %s", *name, err)
	}

	params := &rds.DeleteDBClusterInput{
		DBClusterIdentifier: name,
		SkipFinalSnapshot:   aws.Bool(true),
	}

	_, err = client.DeleteDBCluster(ctx, params)
	return err
}
