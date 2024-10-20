package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (cwdb *CloudWatchDashboards) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var allDashboards []*string

	paginator := cloudwatch.NewListDashboardsPaginator(cwdb.Client, &cloudwatch.ListDashboardsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, dashboard := range page.DashboardEntries {
			if configObj.CloudWatchDashboard.ShouldInclude(config.ResourceValue{
				Name: dashboard.DashboardName,
				Time: dashboard.LastModified,
			}) {
				allDashboards = append(allDashboards, dashboard.DashboardName)
			}
		}
	}

	return allDashboards, nil
}

func (cwdb *CloudWatchDashboards) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No CloudWatch Dashboards to nuke in region %s", cwdb.Region)
		return nil
	}

	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on CloudWatchDashboard.MaxBatchSize, however we add a guard here to warn users when the batching fails and has a
	// chance of throttling AWS. Since we concurrently make one call for each identifier, we pick 100 for the limit here
	// because many APIs in AWS have a limit of 100 requests per second.
	if len(identifiers) > 100 {
		logging.Errorf("Nuking too many CloudWatch Dashboards at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyCloudWatchDashboardsErr{}
	}

	logging.Debugf("Deleting CloudWatch Dashboards in region %s", cwdb.Region)
	input := cloudwatch.DeleteDashboardsInput{DashboardNames: aws.ToStringSlice(identifiers)}
	_, err := cwdb.Client.DeleteDashboards(cwdb.Context, &input)

	// Record status of this resource
	e := report.BatchEntry{
		Identifiers:  aws.ToStringSlice(identifiers),
		ResourceType: "CloudWatch Dashboard",
		Error:        err,
	}
	report.RecordBatch(e)

	if err != nil {
		logging.Debugf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	for _, dashboardName := range identifiers {
		logging.Debugf("[OK] CloudWatch Dashboard %s was deleted in %s", aws.ToString(dashboardName), cwdb.Region)
	}
	return nil
}

// Custom errors

type TooManyCloudWatchDashboardsErr struct{}

func (err TooManyCloudWatchDashboardsErr) Error() string {
	return "Too many CloudWatch Dashboards requested at once."
}
