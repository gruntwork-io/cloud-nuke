package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// CloudWatchDashboard - represents all CloudWatch Dashboards that should be deleted.
type CloudWatchDashboard struct {
	Client         cloudwatchiface.CloudWatchAPI
	Region         string
	DashboardNames []string
}

// ResourceName - the simple name of the aws resource
func (cwdb CloudWatchDashboard) ResourceName() string {
	return "cloudwatch-dashboard"
}

// ResourceIdentifiers - The dashboard names of the cloudwatch dashboards
func (cwdb CloudWatchDashboard) ResourceIdentifiers() []string {
	return cwdb.DashboardNames
}

func (cwdb CloudWatchDashboard) MaxBatchSize() int {
	return 49
}

// Nuke - nuke 'em all!!!
func (cwdb CloudWatchDashboard) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllCloudWatchDashboards(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
