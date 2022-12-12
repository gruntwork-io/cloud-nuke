package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// CloudWatchDashboards - represents all CloudWatch Dashboards that should be deleted.
type CloudWatchDashboards struct {
	DashboardNames []string
}

// ResourceName - the simple name of the aws resource
func (cwdb CloudWatchDashboards) ResourceName() string {
	return "cloudwatch-dashboard"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (cwdb CloudWatchDashboards) ResourceIdentifiers() []string {
	return cwdb.DashboardNames
}

func (cwdb CloudWatchDashboards) MaxBatchSize() int {
	return 49
}

// Nuke - nuke 'em all!!!
func (cwdb CloudWatchDashboards) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllCloudWatchDashboards(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
