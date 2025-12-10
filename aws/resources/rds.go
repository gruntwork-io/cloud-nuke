package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

func (di *DBInstances) getAll(ctx context.Context, configObj config.Config) ([]*string, error) {
	result, err := di.Client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string

	for _, database := range result.DBInstances {
		if configObj.DBInstances.ShouldInclude(config.ResourceValue{
			Time: database.InstanceCreateTime,
			Name: database.DBInstanceIdentifier,
			Tags: util.ConvertRDSTypeTagsToMap(database.TagList),
		}) {
			names = append(names, database.DBInstanceIdentifier)
		}
	}

	return names, nil
}

func (di *DBInstances) nukeAll(names []*string) error {
	if len(names) == 0 {
		logging.Debugf("No RDS DB Instance to nuke in region %s", di.Region)
		return nil
	}

	logging.Debugf("Deleting all RDS Instances in region %s", di.Region)
	deletedNames := []*string{}

	for _, name := range names {
		// Check if instance is part of a cluster before trying to disable deletion protection
		describeResp, err := di.Client.DescribeDBInstances(di.Context, &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: name,
		})
		if err != nil {
			logging.Warnf("[Failed] to describe instance %s: %s", *name, err)
			continue
		}
		// Only disable deletion protection if instance is not part of a cluster
		if len(describeResp.DBInstances) > 0 && describeResp.DBInstances[0].DBClusterIdentifier == nil {
			_, modifyErr := di.Client.ModifyDBInstance(context.TODO(), &rds.ModifyDBInstanceInput{
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

		_, err = di.Client.DeleteDBInstance(di.Context, params)

		if err != nil {
			logging.Errorf("[Failed] %s: %s", *name, err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Debugf("Deleted RDS DB Instance: %s", aws.ToString(name))
		}
	}

	if len(deletedNames) > 0 {
		for _, name := range deletedNames {
			waiter := rds.NewDBInstanceDeletedWaiter(di.Client)
			err := waiter.Wait(di.Context, &rds.DescribeDBInstancesInput{
				DBInstanceIdentifier: name,
			}, di.Timeout)

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

	logging.Debugf("[OK] %d RDS DB Instance(s) deleted in %s", len(deletedNames), di.Region)
	return nil
}
