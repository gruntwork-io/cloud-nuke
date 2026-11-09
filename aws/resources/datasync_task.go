package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/datasync"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// DataSyncTaskAPI defines the interface for DataSync Task operations.
type DataSyncTaskAPI interface {
	DeleteTask(ctx context.Context, params *datasync.DeleteTaskInput, optFns ...func(*datasync.Options)) (*datasync.DeleteTaskOutput, error)
	DescribeTask(ctx context.Context, params *datasync.DescribeTaskInput, optFns ...func(*datasync.Options)) (*datasync.DescribeTaskOutput, error)
	ListTasks(ctx context.Context, params *datasync.ListTasksInput, optFns ...func(*datasync.Options)) (*datasync.ListTasksOutput, error)
	ListTagsForResource(ctx context.Context, params *datasync.ListTagsForResourceInput, optFns ...func(*datasync.Options)) (*datasync.ListTagsForResourceOutput, error)
}

// NewDataSyncTask creates a new DataSync Task resource using the generic resource pattern.
func NewDataSyncTask() AwsResource {
	return NewAwsResource(&resource.Resource[DataSyncTaskAPI]{
		ResourceTypeName: "data-sync-task",
		// DataSync API limit is 20 requests; using 19 to stay safely under the limit.
		BatchSize: 19,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[DataSyncTaskAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = datasync.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.DataSyncTask
		},
		Lister: listDataSyncTasks,
		Nuker:  resource.SimpleBatchDeleter(deleteDataSyncTask),
	})
}

// listDataSyncTasks retrieves all DataSync tasks that match the config filters.
func listDataSyncTasks(ctx context.Context, client DataSyncTaskAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	paginator := datasync.NewListTasksPaginator(client, &datasync.ListTasksInput{
		MaxResults: aws.Int32(100),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, task := range page.Tasks {
			var tags map[string]string
			tagsOutput, err := client.ListTagsForResource(ctx, &datasync.ListTagsForResourceInput{
				ResourceArn: task.TaskArn,
			})
			if err != nil {
				logging.Debugf("Error getting tags for DataSync task %s: %v", aws.ToString(task.TaskArn), err)
				continue
			}
			if tagsOutput != nil {
				tags = util.ConvertDataSyncTagsToMap(tagsOutput.Tags)
			}

			var createdAt *time.Time
			if cfg.HasTimeFilter() {
				createdAt = describeDataSyncTaskCreationTime(ctx, client, task.TaskArn)
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: task.Name,
				Time: createdAt,
				Tags: tags,
			}) {
				identifiers = append(identifiers, task.TaskArn)
			}
		}
	}

	return identifiers, nil
}

// describeDataSyncTaskCreationTime reads the creation time of a task, which
// ListTasks does not return. It returns nil, and warns, when the time cannot be
// read, since a time filter then cannot include the task.
func describeDataSyncTaskCreationTime(ctx context.Context, client DataSyncTaskAPI, taskArn *string) *time.Time {
	output, err := client.DescribeTask(ctx, &datasync.DescribeTaskInput{TaskArn: taskArn})
	if err != nil {
		logging.Warnf("Unable to read the creation time of DataSync task %s, so a time filter cannot include it: %v", aws.ToString(taskArn), err)
		return nil
	}
	return output.CreationTime
}

// deleteDataSyncTask deletes a single DataSync task.
func deleteDataSyncTask(ctx context.Context, client DataSyncTaskAPI, taskArn *string) error {
	_, err := client.DeleteTask(ctx, &datasync.DeleteTaskInput{
		TaskArn: taskArn,
	})
	return err
}
