package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func getAllRdsInstances(session *session.Session, excludeAfter time.Time) ([]*string, error) {
	svc := rds.New(session)

	result, err := svc.DescribeDBInstances(&rds.DescribeDBInstancesInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string

	for _, database := range result.DBInstances {
		if database.InstanceCreateTime != nil && excludeAfter.After(awsgo.TimeValue(database.InstanceCreateTime)) && !hasRDSExcludeTag(database) {
			names = append(names, database.DBInstanceIdentifier)
		} else if !hasRDSExcludeTag(database) {
			names = append(names, database.DBInstanceIdentifier)
		}
	}

	return names, nil
}

// hasRDSExcludeTag checks whether the exlude tag is set for a resource to skip deleting it.
func hasRDSExcludeTag(database *rds.DBInstance) bool {
	// Exclude deletion of any RDS with cloud-nuke-excluded tags
	for _, tag := range database.TagList {
		if *tag.Key == AwsResourceExclusionTagKey && *tag.Value == "true" {
			return true
		}
	}
	return false
}

func nukeAllRdsInstances(session *session.Session, names []*string) error {
	svc := rds.New(session)

	if len(names) == 0 {
		logging.Logger.Debugf("No RDS DB Instance to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all RDS Instances in region %s", *session.Config.Region)
	deletedNames := []*string{}

	for _, name := range names {
		params := &rds.DeleteDBInstanceInput{
			DBInstanceIdentifier: name,
			SkipFinalSnapshot:    awsgo.Bool(true),
		}

		_, err := svc.DeleteDBInstance(params)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s: %s", *name, err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Logger.Debugf("Deleted RDS DB Instance: %s", awsgo.StringValue(name))
		}
	}

	if len(deletedNames) > 0 {
		for _, name := range deletedNames {

			err := svc.WaitUntilDBInstanceDeleted(&rds.DescribeDBInstancesInput{
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
				logging.Logger.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}

	logging.Logger.Debugf("[OK] %d RDS DB Instance(s) deleted in %s", len(deletedNames), *session.Config.Region)
	return nil
}
