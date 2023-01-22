package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/gruntwork-io/go-commons/errors"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
)

func getAllCloudWatchDashboards(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := cloudwatch.New(session)

	allDashboards := []*string{}
	input := &cloudwatch.ListDashboardsInput{}
	err := svc.ListDashboardsPages(
		input,
		func(page *cloudwatch.ListDashboardsOutput, lastPage bool) bool {
			for _, dashboard := range page.DashboardEntries {
				if shouldIncludeCloudWatchDashboard(dashboard, excludeAfter, configObj) {
					allDashboards = append(allDashboards, dashboard.DashboardName)
				}
			}
			return !lastPage
		},
	)
	return allDashboards, errors.WithStackTrace(err)
}

func shouldIncludeCloudWatchDashboard(dashboard *cloudwatch.DashboardEntry, excludeAfter time.Time, configObj config.Config) bool {
	if dashboard == nil {
		return false
	}

	if dashboard.LastModified != nil && excludeAfter.Before(*dashboard.LastModified) {
		return false
	}

	return config.ShouldInclude(
		aws.StringValue(dashboard.DashboardName),
		configObj.CloudWatchDashboard.IncludeRule.NamesRegExp,
		configObj.CloudWatchDashboard.ExcludeRule.NamesRegExp,
	)
}

func nukeAllCloudWatchDashboards(session *session.Session, identifiers []*string) error {
	region := aws.StringValue(session.Config.Region)

	svc := cloudwatch.New(session)

	if len(identifiers) == 0 {
		logging.Logger.Debugf("No CloudWatch Dashboards to nuke in region %s", region)
		return nil
	}

	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on CloudWatchDashboard.MaxBatchSize, however we add a guard here to warn users when the batching fails and has a
	// chance of throttling AWS. Since we concurrently make one call for each identifier, we pick 100 for the limit here
	// because many APIs in AWS have a limit of 100 requests per second.
	if len(identifiers) > 100 {
		logging.Logger.Errorf("Nuking too many CloudWatch Dashboards at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyCloudWatchDashboardsErr{}
	}

	logging.Logger.Debugf("Deleting CloudWatch Dashboards in region %s", region)
	input := cloudwatch.DeleteDashboardsInput{DashboardNames: identifiers}
	_, err := svc.DeleteDashboards(&input)

	// Record status of this resource
	e := report.BatchEntry{
		Identifiers:  aws.StringValueSlice(identifiers),
		ResourceType: "CloudWatch Dashboard",
		Error:        err,
	}
	report.RecordBatch(e)

	if err != nil {
		logging.Logger.Debugf("[Failed] %s", err)
		return errors.WithStackTrace(err)
	}

	for _, dashboardName := range identifiers {
		logging.Logger.Debugf("[OK] CloudWatch Dashboard %s was deleted in %s", aws.StringValue(dashboardName), region)
	}
	return nil
}

// Custom errors

type TooManyCloudWatchDashboardsErr struct{}

func (err TooManyCloudWatchDashboardsErr) Error() string {
	return "Too many CloudWatch Dashboards requested at once."
}
