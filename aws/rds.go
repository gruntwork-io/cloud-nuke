package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

func (di *DBInstances) getAll(configObj config.Config) ([]*string, error) {
	result, err := di.Client.DescribeDBInstances(&rds.DescribeDBInstancesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string

	for _, database := range result.DBInstances {
		if configObj.DBInstances.ShouldInclude(config.ResourceValue{
			Time: database.InstanceCreateTime,
			Name: database.DBName,
		}) {
			names = append(names, database.DBInstanceIdentifier)
		}
	}

	return names, nil
}

func (di *DBInstances) nukeAll(names []*string) error {
	if len(names) == 0 {
		logging.Logger.Debugf("No RDS DB Instance to nuke in region %s", di.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all RDS Instances in region %s", di.Region)
	deletedNames := []*string{}

	for _, name := range names {
		params := &rds.DeleteDBInstanceInput{
			DBInstanceIdentifier: name,
			SkipFinalSnapshot:    awsgo.Bool(true),
		}

		_, err := di.Client.DeleteDBInstance(params)

		if err != nil {
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking RDS Instance",
			}, map[string]interface{}{
				"region": di.Region,
			})
			logging.Logger.Errorf("[Failed] %s: %s", *name, err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Logger.Debugf("Deleted RDS DB Instance: %s", awsgo.StringValue(name))
		}
	}

	if len(deletedNames) > 0 {
		for _, name := range deletedNames {

			err := di.Client.WaitUntilDBInstanceDeleted(&rds.DescribeDBInstancesInput{
				DBInstanceIdentifier: name,
			})

			// Record status of this resource
			e := report.Entry{
				Identifier:   aws.StringValue(name),
				ResourceType: "RDS Instance",
				Error:        err,
			}
			report.Record(e)

			if err != nil {
				telemetry.TrackEvent(commonTelemetry.EventContext{
					EventName: "Error Nuking RDS Instance",
				}, map[string]interface{}{
					"region": di.Region,
				})
				logging.Logger.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}

	logging.Logger.Debugf("[OK] %d RDS DB Instance(s) deleted in %s", len(deletedNames), di.Region)
	return nil
}
