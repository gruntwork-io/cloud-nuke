package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// NewCloudWatchLogGroups creates a new CloudWatch Log Groups resource using the generic resource pattern.
func NewCloudWatchLogGroups() AwsResource {
	return NewAwsResource(&resource.Resource[*cloudwatchlogs.Client]{
		ResourceTypeName: "cloudwatch-loggroup",
		// Tentative batch size to ensure AWS doesn't throttle. Note that CloudWatch Logs does not support bulk delete,
		// so we will be deleting this many in parallel using go routines. We pick 35 here, which is half of what the
		// AWS web console will do. We pick a conservative number here to avoid hitting AWS API rate limits.
		BatchSize: 35,
		InitClient: func(r *resource.Resource[*cloudwatchlogs.Client], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for CloudWatchLogs client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = cloudwatchlogs.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.CloudWatchLogGroup
		},
		Lister: listCloudWatchLogGroups,
		Nuker:  resource.SimpleBatchDeleter(deleteCloudWatchLogGroup),
	})
}

// listCloudWatchLogGroups retrieves all CloudWatch Log Groups that match the config filters.
func listCloudWatchLogGroups(ctx context.Context, client *cloudwatchlogs.Client, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var allLogGroups []*string

	paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(client, &cloudwatchlogs.DescribeLogGroupsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, logGroup := range page.LogGroups {
			var creationTime *time.Time
			if logGroup.CreationTime != nil {
				// Convert milliseconds since epoch to time.Time object
				creationTime = aws.Time(time.Unix(0, aws.ToInt64(logGroup.CreationTime)*int64(time.Millisecond)))
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: logGroup.LogGroupName,
				Time: creationTime,
			}) {
				allLogGroups = append(allLogGroups, logGroup.LogGroupName)
			}
		}
	}

	return allLogGroups, nil
}

// deleteCloudWatchLogGroup deletes a single CloudWatch Log Group.
func deleteCloudWatchLogGroup(ctx context.Context, client *cloudwatchlogs.Client, logGroupName *string) error {
	_, err := client.DeleteLogGroup(ctx, &cloudwatchlogs.DeleteLogGroupInput{
		LogGroupName: logGroupName,
	})
	if err != nil {
		return err
	}

	logging.Debugf("Deleted CloudWatch Log Group: %s", aws.ToString(logGroupName))
	return nil
}
