package resources

import (
	"context"
	goerr "errors"
	"time"

	"github.com/gruntwork-io/cloud-nuke/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (dsg *DBSubnetGroups) waitUntilRdsDbSubnetGroupDeleted(name *string) error {
	// wait up to 15 minutes
	for i := 0; i < 90; i++ {
		_, err := dsg.Client.DescribeDBSubnetGroups(
			dsg.Context, &rds.DescribeDBSubnetGroupsInput{DBSubnetGroupName: name})
		if err != nil {
			var notFoundErr *types.DBSubnetGroupNotFoundFault
			if goerr.As(err, &notFoundErr) {
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
	paginator := rds.NewDescribeDBSubnetGroupsPaginator(dsg.Client, &rds.DescribeDBSubnetGroupsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, subnetGroup := range page.DBSubnetGroups {
			if configObj.DBSubnetGroups.ShouldInclude(config.ResourceValue{
				Name: subnetGroup.DBSubnetGroupName,
			}) {
				names = append(names, subnetGroup.DBSubnetGroupName)
			}
		}
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
		_, err := dsg.Client.DeleteDBSubnetGroup(dsg.Context, &rds.DeleteDBSubnetGroupInput{
			DBSubnetGroupName: name,
		})

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(name),
			ResourceType: "RDS DB Subnet Group",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s: %s", *name, err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Debugf("Deleted RDS DB subnet group: %s", aws.ToString(name))
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
