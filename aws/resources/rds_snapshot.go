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

// RdsSnapshotAPI defines the interface for RDS Snapshot operations.
type RdsSnapshotAPI interface {
	DescribeDBSnapshots(ctx context.Context, params *rds.DescribeDBSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSnapshotsOutput, error)
	DeleteDBSnapshot(ctx context.Context, params *rds.DeleteDBSnapshotInput, optFns ...func(*rds.Options)) (*rds.DeleteDBSnapshotOutput, error)
}

// NewRdsSnapshot creates a new RDS Snapshot resource using the generic resource pattern.
func NewRdsSnapshot() AwsResource {
	return NewAwsResource(&resource.Resource[RdsSnapshotAPI]{
		ResourceTypeName: "rds-snapshot",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[RdsSnapshotAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = rds.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.RdsSnapshot
		},
		Lister: listRdsSnapshots,
		Nuker:  resource.SimpleBatchDeleter(deleteRdsSnapshot),
	})
}

// listRdsSnapshots retrieves all RDS snapshots that match the config filters.
func listRdsSnapshots(ctx context.Context, client RdsSnapshotAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	paginator := rds.NewDescribeDBSnapshotsPaginator(client, &rds.DescribeDBSnapshotsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, s := range page.DBSnapshots {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: s.DBSnapshotIdentifier,
				Time: s.SnapshotCreateTime,
				Tags: util.ConvertRDSTypeTagsToMap(s.TagList),
			}) {
				identifiers = append(identifiers, s.DBSnapshotIdentifier)
			}
		}
	}

	return identifiers, nil
}

// deleteRdsSnapshot deletes a single RDS snapshot.
func deleteRdsSnapshot(ctx context.Context, client RdsSnapshotAPI, identifier *string) error {
	_, err := client.DeleteDBSnapshot(ctx, &rds.DeleteDBSnapshotInput{
		DBSnapshotIdentifier: identifier,
	})
	return err
}
