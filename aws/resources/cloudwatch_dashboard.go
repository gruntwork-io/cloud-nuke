package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// CloudWatchDashboardsAPI is the interface for CloudWatch dashboard operations.
type CloudWatchDashboardsAPI interface {
	ListDashboards(ctx context.Context, params *cloudwatch.ListDashboardsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.ListDashboardsOutput, error)
	DeleteDashboards(ctx context.Context, params *cloudwatch.DeleteDashboardsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DeleteDashboardsOutput, error)
}

// NewCloudWatchDashboards creates a new CloudWatch Dashboards resource using the generic resource pattern.
func NewCloudWatchDashboards() AwsResource {
	return NewAwsResource(&resource.Resource[*cloudwatch.Client]{
		ResourceTypeName: "cloudwatch-dashboard",
		BatchSize:        49,
		InitClient: func(r *resource.Resource[*cloudwatch.Client], cfg any) {
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
		Nuker:  deleteCloudWatchDashboards,
	})
}

// listCloudWatchDashboards retrieves all CloudWatch dashboards that match the config filters.
func listCloudWatchDashboards(ctx context.Context, client *cloudwatch.Client, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	return listCloudWatchDashboardsWithClient(ctx, client, cfg)
}

// listCloudWatchDashboardsWithClient is the internal implementation that accepts an interface for testability.
func listCloudWatchDashboardsWithClient(ctx context.Context, client cloudwatch.ListDashboardsAPIClient, cfg config.ResourceType) ([]*string, error) {
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
func deleteCloudWatchDashboards(ctx context.Context, client *cloudwatch.Client, scope resource.Scope, resourceType string, identifiers []*string) error {
	return deleteCloudWatchDashboardsWithClient(ctx, client, scope, resourceType, identifiers)
}

// DeleteDashboardsAPI is the interface for deleting CloudWatch dashboards.
type DeleteDashboardsAPI interface {
	DeleteDashboards(ctx context.Context, params *cloudwatch.DeleteDashboardsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DeleteDashboardsOutput, error)
}

// deleteCloudWatchDashboardsWithClient is the internal implementation that accepts an interface for testability.
func deleteCloudWatchDashboardsWithClient(ctx context.Context, client DeleteDashboardsAPI, scope resource.Scope, resourceType string, identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No %s to nuke in %s", resourceType, scope)
		return nil
	}

	// Guard against too many requests that could cause rate limiting
	if len(identifiers) > 100 {
		logging.Errorf("Nuking too many %s at once (%d): halting to avoid hitting rate limiting",
			resourceType, len(identifiers))
		return fmt.Errorf("too many %s requested at once", resourceType)
	}

	logging.Debugf("Deleting %d %s in %s", len(identifiers), resourceType, scope)

	// Use the bulk delete API
	input := &cloudwatch.DeleteDashboardsInput{
		DashboardNames: aws.ToStringSlice(identifiers),
	}
	_, err := client.DeleteDashboards(ctx, input)

	// Record status of this resource using batch entry
	e := report.BatchEntry{
		Identifiers:  aws.ToStringSlice(identifiers),
		ResourceType: resourceType,
		Error:        err,
	}
	report.RecordBatch(e)

	if err != nil {
		logging.Debugf("[Failed] %s", err)
		return err
	}

	for _, dashboardName := range identifiers {
		logging.Debugf("[OK] CloudWatch Dashboard %s was deleted in %s", aws.ToString(dashboardName), scope)
	}
	return nil
}
