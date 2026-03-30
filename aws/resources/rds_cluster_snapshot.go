package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// RdsClusterSnapshotAPI defines the interface for RDS Cluster Snapshot operations.
type RdsClusterSnapshotAPI interface {
	DescribeDBClusterSnapshots(ctx context.Context, params *rds.DescribeDBClusterSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClusterSnapshotsOutput, error)
	DeleteDBClusterSnapshot(ctx context.Context, params *rds.DeleteDBClusterSnapshotInput, optFns ...func(*rds.Options)) (*rds.DeleteDBClusterSnapshotOutput, error)
}

// NewRdsClusterSnapshot creates a new RDS Cluster Snapshot resource using the generic resource pattern.
func NewRdsClusterSnapshot() AwsResource {
	return NewAwsResource(&resource.Resource[RdsClusterSnapshotAPI]{
		ResourceTypeName: "rds-cluster-snapshot",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[RdsClusterSnapshotAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = rds.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.RDSClusterSnapshot
		},
		Lister: listRdsClusterSnapshots,
		Nuker:  resource.SimpleBatchDeleter(deleteRdsClusterSnapshot),
	})
}

// listRdsClusterSnapshots retrieves all RDS cluster snapshots that match the config filters.
func listRdsClusterSnapshots(ctx context.Context, client RdsClusterSnapshotAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	paginator := rds.NewDescribeDBClusterSnapshotsPaginator(client, &rds.DescribeDBClusterSnapshotsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, s := range page.DBClusterSnapshots {
			// Automated snapshots are managed by AWS and cannot be manually deleted
			if aws.ToString(s.SnapshotType) == "automated" {
				continue
			}
			// Only snapshots in "available" or "failed" state can be deleted
			status := aws.ToString(s.Status)
			if status != "available" && status != "failed" {
				continue
			}
			if cfg.ShouldInclude(config.ResourceValue{
				Name: s.DBClusterSnapshotIdentifier,
				Time: s.SnapshotCreateTime,
				Tags: util.ConvertRDSTypeTagsToMap(s.TagList),
			}) {
				identifiers = append(identifiers, s.DBClusterSnapshotIdentifier)
			}
		}
	}

	return identifiers, nil
}

// deleteRdsClusterSnapshot deletes a single RDS cluster snapshot.
func deleteRdsClusterSnapshot(ctx context.Context, client RdsClusterSnapshotAPI, identifier *string) error {
	_, err := client.DeleteDBClusterSnapshot(ctx, &rds.DeleteDBClusterSnapshotInput{
		DBClusterSnapshotIdentifier: identifier,
	})
	return err
}
