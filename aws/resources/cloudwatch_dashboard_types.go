package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type CloudWatchDashboardsAPI interface {
	ListDashboards(ctx context.Context, params *cloudwatch.ListDashboardsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.ListDashboardsOutput, error)
	DeleteDashboards(ctx context.Context, params *cloudwatch.DeleteDashboardsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DeleteDashboardsOutput, error)
}

// CloudWatchDashboards - represents all CloudWatch Dashboards that should be deleted.
type CloudWatchDashboards struct {
	BaseAwsResource
	Client         CloudWatchDashboardsAPI
	Region         string
	DashboardNames []string
}

func (cwdb *CloudWatchDashboards) InitV2(cfg aws.Config) {
	cwdb.Client = cloudwatch.NewFromConfig(cfg)
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

func (cwdb *CloudWatchDashboards) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.CloudWatchDashboard
}
func (cwdb *CloudWatchDashboards) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := cwdb.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	cwdb.DashboardNames = aws.ToStringSlice(identifiers)
	return cwdb.DashboardNames, nil
}

// Nuke - nuke 'em all!!!
func (cwdb *CloudWatchDashboards) Nuke(identifiers []string) error {
	if err := cwdb.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
