package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// DBInstancesAPI defines the interface for RDS DB Instance operations.
type DBInstancesAPI interface {
	ModifyDBInstance(ctx context.Context, params *rds.ModifyDBInstanceInput, optFns ...func(*rds.Options)) (*rds.ModifyDBInstanceOutput, error)
	DeleteDBInstance(ctx context.Context, params *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error)
	DescribeDBInstances(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error)
}

// NewDBInstances creates a new DBInstances resource using the generic resource pattern.
func NewDBInstances() AwsResource {
	return NewAwsResource(&resource.Resource[DBInstancesAPI]{
		ResourceTypeName: "rds",
		BatchSize:        49,
		InitClient: func(r *resource.Resource[DBInstancesAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for RDS client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = rds.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.DBInstances.ResourceType
		},
		Lister: listDBInstances,
		Nuker:  deleteDBInstances,
	})
}

// listDBInstances retrieves all RDS DB instances that match the config filters.
func listDBInstances(ctx context.Context, client DBInstancesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	result, err := client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string
	for _, database := range result.DBInstances {
		if cfg.ShouldInclude(config.ResourceValue{
			Time: database.InstanceCreateTime,
			Name: database.DBInstanceIdentifier,
			Tags: util.ConvertRDSTypeTagsToMap(database.TagList),
		}) {
			names = append(names, database.DBInstanceIdentifier)
		}
	}

	return names, nil
}

// deleteDBInstances deletes all RDS DB instances.
func deleteDBInstances(ctx context.Context, client DBInstancesAPI, scope resource.Scope, resourceType string, names []*string) error {
	if len(names) == 0 {
		logging.Debugf("No RDS DB Instance to nuke in region %s", scope.Region)
		return nil
	}

	logging.Debugf("Deleting all RDS Instances in region %s", scope.Region)
	deletedNames := []*string{}

	for _, name := range names {
		// Check if instance is part of a cluster before trying to disable deletion protection
		describeResp, err := client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: name,
		})
		if err != nil {
			logging.Warnf("[Failed] to describe instance %s: %s", *name, err)
			continue
		}
		// Only disable deletion protection if instance is not part of a cluster
		if len(describeResp.DBInstances) > 0 && describeResp.DBInstances[0].DBClusterIdentifier == nil {
			_, modifyErr := client.ModifyDBInstance(ctx, &rds.ModifyDBInstanceInput{
				DBInstanceIdentifier: name,
				DeletionProtection:   aws.Bool(false),
				ApplyImmediately:     aws.Bool(true),
			})
			if modifyErr != nil {
				logging.Warnf("[Failed] to disable deletion protection for %s: %s", *name, modifyErr)
			}
		} else {
			logging.Debugf("Skipping deletion protection modification for cluster member instance %s", *name)
		}

		params := &rds.DeleteDBInstanceInput{
			DBInstanceIdentifier: name,
			SkipFinalSnapshot:    aws.Bool(true),
		}

		_, err = client.DeleteDBInstance(ctx, params)

		if err != nil {
			logging.Errorf("[Failed] %s: %s", *name, err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Debugf("Deleted RDS DB Instance: %s", aws.ToString(name))
		}
	}

	if len(deletedNames) > 0 {
		for _, name := range deletedNames {
			waiter := rds.NewDBInstanceDeletedWaiter(client)
			err := waiter.Wait(ctx, &rds.DescribeDBInstancesInput{
				DBInstanceIdentifier: name,
			}, 5*time.Minute)

			// Record status of this resource
			e := report.Entry{
				Identifier:   aws.ToString(name),
				ResourceType: "RDS Instance",
				Error:        err,
			}
			report.Record(e)

			if err != nil {
				logging.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}

	logging.Debugf("[OK] %d RDS DB Instance(s) deleted in %s", len(deletedNames), scope.Region)
	return nil
}

// RdsDeleteError represents an error when deleting RDS resources.
type RdsDeleteError struct {
	name string
}

func (e RdsDeleteError) Error() string {
	return "RDS DB Instance:" + e.name + "was not deleted"
}
