package aws

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

func waitUntilRdsDbSubnetGroupDeleted(svc *rds.RDS, name *string) error {
	// wait up to 15 minutes
	for i := 0; i < 90; i++ {
		_, err := svc.DescribeDBSubnetGroups(&rds.DescribeDBSubnetGroupsInput{DBSubnetGroupName: name})
		if err != nil {
			if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == rds.ErrCodeDBSubnetGroupNotFoundFault {
				return nil
			}

			return err
		}

		time.Sleep(10 * time.Second)
		logging.Logger.Debug("Waiting for RDS Cluster to be deleted")
	}

	return RdsDeleteError{name: *name}
}

func getAllRdsDbSubnetGroups(session *session.Session) ([]*string, error) {
	svc := rds.New(session)

	result, err := svc.DescribeDBSubnetGroups(&rds.DescribeDBSubnetGroupsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string

	for _, sg := range result.DBSubnetGroups {
		names = append(names, sg.DBSubnetGroupName)
	}

	return names, nil
}

func nukeAllRdsDbSubnetGroups(session *session.Session, names []*string) error {
	svc := rds.New(session)

	if len(names) == 0 {
		logging.Logger.Debugf("No DB Subnet groups in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all DB Subnet groups in region %s", *session.Config.Region)
	deletedNames := []*string{}

	for _, name := range names {
		_, err := svc.DeleteDBSubnetGroup(&rds.DeleteDBSubnetGroupInput{
			DBSubnetGroupName: name,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(name),
			ResourceType: "RDS DB Subnet Group",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("[Failed] %s: %s", *name, err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking RDS DB subnet group",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
		} else {
			deletedNames = append(deletedNames, name)
			logging.Logger.Debugf("Deleted RDS DB subnet group: %s", awsgo.StringValue(name))
		}
	}

	if len(deletedNames) > 0 {
		for _, name := range deletedNames {

			err := waitUntilRdsDbSubnetGroupDeleted(svc, name)
			if err != nil {
				logging.Logger.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}

	logging.Logger.Debugf("[OK] %d RDS DB subnet group(s) nuked in %s", len(deletedNames), *session.Config.Region)
	return nil
}
