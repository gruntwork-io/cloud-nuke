package resources

import (
	"context"

	"github.com/andrewderr/cloud-nuke-a1/config"
	"github.com/andrewderr/cloud-nuke-a1/logging"
	"github.com/andrewderr/cloud-nuke-a1/report"
	"github.com/andrewderr/cloud-nuke-a1/telemetry"
	"github.com/andrewderr/cloud-nuke-a1/util"
	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

func (di *DBInstances) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := di.Client.DescribeDBInstances(&rds.DescribeDBInstancesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string

	for _, database := range result.DBInstances {
		if configObj.DBInstances.ShouldInclude(config.ResourceValue{
			Time: database.InstanceCreateTime,
			Name: database.DBName,
			Tags: util.ConvertRDSTagsToMap(database.TagList),
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
			logging.Errorf("[Failed] %s: %s", *name, err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Debugf("Deleted RDS DB Instance: %s", awsgo.StringValue(name))
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
				logging.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}

	logging.Debugf("[OK] %d RDS DB Instance(s) deleted in %s", len(deletedNames), di.Region)
	return nil
}
