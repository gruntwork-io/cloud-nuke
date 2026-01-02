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

// DBInstancesAPI defines the interface for RDS DB Instance operations.
type DBInstancesAPI interface {
	DescribeDBInstances(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error)
	ModifyDBInstance(ctx context.Context, params *rds.ModifyDBInstanceInput, optFns ...func(*rds.Options)) (*rds.ModifyDBInstanceOutput, error)
	DeleteDBInstance(ctx context.Context, params *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error)
}

// NewDBInstances creates a new DBInstances resource using the generic resource pattern.
func NewDBInstances() AwsResource {
	return NewAwsResource(&resource.Resource[DBInstancesAPI]{
		ResourceTypeName: "rds",
		BatchSize:        49,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[DBInstancesAPI], cfg aws.Config) {
			r.Client = rds.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.DBInstances.ResourceType
		},
		Lister: listDBInstances,
		Nuker:  resource.SequentialDeleteThenWaitAll(deleteDBInstance, waitForDBInstancesDeleted),
	})
}

// listDBInstances retrieves all RDS DB instances that match the config filters.
func listDBInstances(ctx context.Context, client DBInstancesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var names []*string

	paginator := rds.NewDescribeDBInstancesPaginator(client, &rds.DescribeDBInstancesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, db := range page.DBInstances {
			if cfg.ShouldInclude(config.ResourceValue{
				Time: db.InstanceCreateTime,
				Name: db.DBInstanceIdentifier,
				Tags: util.ConvertRDSTypeTagsToMap(db.TagList),
			}) {
				names = append(names, db.DBInstanceIdentifier)
			}
		}
	}

	return names, nil
}

// deleteDBInstance deletes a single RDS DB instance.
// For standalone instances, it first disables deletion protection.
// For cluster members, deletion protection is managed at the cluster level.
func deleteDBInstance(ctx context.Context, client DBInstancesAPI, name *string) error {
	resp, err := client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: name,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	if len(resp.DBInstances) == 0 {
		return nil // Instance doesn't exist
	}

	instance := resp.DBInstances[0]
	isStandalone := instance.DBClusterIdentifier == nil

	if isStandalone {
		if _, err := client.ModifyDBInstance(ctx, &rds.ModifyDBInstanceInput{
			DBInstanceIdentifier: name,
			DeletionProtection:   aws.Bool(false),
			ApplyImmediately:     aws.Bool(true),
		}); err != nil {
			logging.Warnf("[Failed] to disable deletion protection for %s: %s", aws.ToString(name), err)
		}
	}

	_, err = client.DeleteDBInstance(ctx, &rds.DeleteDBInstanceInput{
		DBInstanceIdentifier: name,
		SkipFinalSnapshot:    aws.Bool(true),
	})
	return errors.WithStackTrace(err)
}

// waitForDBInstancesDeleted waits for all specified RDS DB instances to be deleted.
func waitForDBInstancesDeleted(ctx context.Context, client DBInstancesAPI, ids []string) error {
	waiter := rds.NewDBInstanceDeletedWaiter(client)
	for _, id := range ids {
		if err := waiter.Wait(ctx, &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(id),
		}, 10*time.Minute); err != nil {
			return err
		}
	}
	return nil
}
