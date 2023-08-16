package resources

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// CloudWatchDashboards - represents all CloudWatch Dashboards that should be deleted.
type CloudWatchDashboards struct {
	Client         cloudwatchiface.CloudWatchAPI
	Region         string
	DashboardNames []string
}

func (cwdb *CloudWatchDashboards) Init(session *session.Session) {
	cwdb.Client = cloudwatch.New(session)
}

// ResourceName - the simple name of the aws resource
func (cwdb *CloudWatchDashboards) ResourceName() string {
	return "cloudwatch-dashboard"
}

// ResourceIdentifiers - The dashboard names of the cloudwatch dashboards
func (cwdb *CloudWatchDashboards) ResourceIdentifiers() []string {
	return cwdb.DashboardNames
}

func (cwdb *CloudWatchDashboards) MaxBatchSize() int {
	return 49
}

func (cwdb *CloudWatchDashboards) GetAndSetIdentifiers(configObj config.Config) ([]string, error) {
	identifiers, err := cwdb.getAll(configObj)
	if err != nil {
		return nil, err
	}

	cwdb.DashboardNames = awsgo.StringValueSlice(identifiers)
	return cwdb.DashboardNames, nil
}

// Nuke - nuke 'em all!!!
func (cwdb *CloudWatchDashboards) Nuke(identifiers []string) error {
	if err := cwdb.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
