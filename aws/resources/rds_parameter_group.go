package resources

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (pg *RdsParameterGroup) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var names []*string

	// Initialize the paginator
	paginator := rds.NewDescribeDBParameterGroupsPaginator(pg.Client, &rds.DescribeDBParameterGroupsInput{})

	// Iterate through the pages
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		// Process each parameter group on the page
		for _, parameterGroup := range page.DBParameterGroups {
			// we can't delete default parameter group
			// Default parameter group names can include a period, such as default.mysql8.0. However, custom parameter group names can't include a period.
			if strings.HasPrefix(aws.ToString(parameterGroup.DBParameterGroupName), "default.") {
				logging.Debugf("Skipping %s since it is a default parameter group", aws.ToString(parameterGroup.DBParameterGroupName))
				continue
			}

			if configObj.RdsParameterGroup.ShouldInclude(config.ResourceValue{
				Name: parameterGroup.DBParameterGroupName,
			}) {
				names = append(names, parameterGroup.DBParameterGroupName)
			}
		}
	}

	return names, nil
}

func (pg *RdsParameterGroup) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No DB parameter groups in region %s", pg.Region)
		return nil
	}

	logging.Debugf("Deleting all DB parameter groups in region %s", pg.Region)
	var deleted []*string

	for _, identifier := range identifiers {
		logging.Debugf("[RDS Parameter Group] Deleting %s in region %s", *identifier, pg.Region)

		_, err := pg.Client.DeleteDBParameterGroup(
			pg.Context,
			&rds.DeleteDBParameterGroupInput{
				DBParameterGroupName: identifier,
			})
		if err != nil {
			logging.Errorf("[RDS Parameter Group] Error deleting RDS Parameter Group %s: %s", *identifier, err)
		} else {
			deleted = append(deleted, identifier)
			logging.Debugf("[RDS Parameter Group] Deleted RDS Parameter Group %s", *identifier)
		}

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(identifier),
			ResourceType: pg.ResourceName(),
			Error:        err,
		}
		report.Record(e)
	}

	logging.Debugf("[OK] %d RDS DB parameter group(s) nuked in %s", len(deleted), pg.Region)
	return nil
}
