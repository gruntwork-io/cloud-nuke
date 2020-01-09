package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

func waitUntilRdsDeleted(svc *rds.RDS, input *rds.DescribeDBInstancesInput) error {
	for i := 0; i < 240; i++ {
		_, err := svc.DescribeDBInstances(input)
		if err != nil {
			if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == rds.ErrCodeDBInstanceNotFoundFault {
				return nil
			}

			return err
		}

		time.Sleep(1 * time.Second)
		logging.Logger.Debug("Waiting for RDS DB Instance to be deleted")
	}

	return RdsDeleteError{}
}

func getAllRdsInstances(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := rds.New(session)

	result, err := svc.DescribeDBInstances(&rds.DescribeDBInstancesInput{})

	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string

	for _, database := range result.DBInstances {
		//if excludeAfter.After(*database.InstanceCreateTime) {
		names = append(names, database.DBInstanceIdentifier)
		//}
	}

	return names, nil
}

func nukeAllRdsInstances(session *session.Session, names []*string) error {
	svc := rds.New(session)

	if len(names) == 0 {
		logging.Logger.Infof("No RDS DB Instanceto nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all RDS Instances in region %s", *session.Config.Region)
	deletedNames := []*string{}

	for _, name := range names {
		params := &rds.DeleteDBInstanceInput{
			DBInstanceIdentifier: name,
			SkipFinalSnapshot:    awsgo.Bool(true),
		}

		_, err := svc.DeleteDBInstance(params)

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Logger.Infof("Deleted RDS DB Instance: %s", awsgo.StringValue(name))
		}
	}

	if len(deletedNames) > 0 {
		for _, name := range deletedNames {

			err := waitUntilRdsDeleted(svc, &rds.DescribeDBInstancesInput{
				DBInstanceIdentifier: name,
			})

			if err != nil {
				logging.Logger.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}

	logging.Logger.Infof("[OK] %d RDS DB Instance(s) deleted in %s", len(deletedNames), *session.Config.Region)
	return nil
}
