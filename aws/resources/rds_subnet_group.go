package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/gruntwork-io/cloud-nuke/config"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (dsg *DBSubnetGroups) waitUntilRdsDbSubnetGroupDeleted(name *string) error {
	// wait up to 15 minutes
	for i := 0; i < 90; i++ {
		_, err := dsg.Client.DescribeDBSubnetGroups(&rds.DescribeDBSubnetGroupsInput{DBSubnetGroupName: name})
		if err != nil {
			if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == rds.ErrCodeDBSubnetGroupNotFoundFault {
				return nil
			}

			return err
		}

		time.Sleep(10 * time.Second)
		logging.Debug("Waiting for RDS Cluster to be deleted")
	}

	return RdsDeleteError{name: *name}
}

func (dsg *DBSubnetGroups) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var names []*string
	err := dsg.Client.DescribeDBSubnetGroupsPages(
		&rds.DescribeDBSubnetGroupsInput{},
		func(page *rds.DescribeDBSubnetGroupsOutput, lastPage bool) bool {
			for _, subnetGroup := range page.DBSubnetGroups {
				if configObj.DBSubnetGroups.ShouldInclude(config.ResourceValue{
					Name: subnetGroup.DBSubnetGroupName,
				}) {
					names = append(names, subnetGroup.DBSubnetGroupName)
				}
			}

			return !lastPage
		})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return names, nil
}

func (dsg *DBSubnetGroups) nukeAll(names []*string) error {
	if len(names) == 0 {
		logging.Debugf("No DB Subnet groups in region %s", dsg.Region)
		return nil
	}

	logging.Debugf("Deleting all DB Subnet groups in region %s", dsg.Region)
	deletedNames := []*string{}

	for _, name := range names {
		_, err := dsg.Client.DeleteDBSubnetGroup(&rds.DeleteDBSubnetGroupInput{
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
			logging.Debugf("[Failed] %s: %s", *name, err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Debugf("Deleted RDS DB subnet group: %s", awsgo.StringValue(name))
		}
	}

	if len(deletedNames) > 0 {
		for _, name := range deletedNames {

			err := dsg.waitUntilRdsDbSubnetGroupDeleted(name)
			if err != nil {
				logging.Errorf("[Failed] %s", err)
				return errors.WithStackTrace(err)
			}
		}
	}

	logging.Debugf("[OK] %d RDS DB subnet group(s) nuked in %s", len(deletedNames), dsg.Region)
	return nil
}
