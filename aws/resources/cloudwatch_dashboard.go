package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// CloudWatchDashboardsAPI defines the interface for CloudWatch dashboard operations.
type CloudWatchDashboardsAPI interface {
	ListDashboards(ctx context.Context, params *cloudwatch.ListDashboardsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.ListDashboardsOutput, error)
	DeleteDashboards(ctx context.Context, params *cloudwatch.DeleteDashboardsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DeleteDashboardsOutput, error)
}

// NewCloudWatchDashboards creates a new CloudWatch Dashboards resource using the generic resource pattern.
func NewCloudWatchDashboards() AwsResource {
	return NewAwsResource(&resource.Resource[CloudWatchDashboardsAPI]{
		ResourceTypeName: "cloudwatch-dashboard",
		BatchSize:        49,
		InitClient: func(r *resource.Resource[CloudWatchDashboardsAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for CloudWatch client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = cloudwatch.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.CloudWatchDashboard
		},
		Lister: listCloudWatchDashboards,
		Nuker:  resource.BulkDeleter(deleteCloudWatchDashboards),
	})
}

// listCloudWatchDashboards retrieves all CloudWatch dashboards that match the config filters.
func listCloudWatchDashboards(ctx context.Context, client CloudWatchDashboardsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var allDashboards []*string

	paginator := cloudwatch.NewListDashboardsPaginator(client, &cloudwatch.ListDashboardsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, dashboard := range page.DashboardEntries {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: dashboard.DashboardName,
				Time: dashboard.LastModified,
			}) {
				allDashboards = append(allDashboards, dashboard.DashboardName)
			}
		}
	}

	return allDashboards, nil
}

// deleteCloudWatchDashboards deletes CloudWatch dashboards using the bulk delete API.
func deleteCloudWatchDashboards(ctx context.Context, client CloudWatchDashboardsAPI, ids []string) error {
	_, err := client.DeleteDashboards(ctx, &cloudwatch.DeleteDashboardsInput{
		DashboardNames: ids,
	})
	return err
}
