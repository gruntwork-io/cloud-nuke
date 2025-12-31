package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/datasync"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// DataSyncTaskAPI defines the interface for DataSync Task operations.
type DataSyncTaskAPI interface {
	DeleteTask(ctx context.Context, params *datasync.DeleteTaskInput, optFns ...func(*datasync.Options)) (*datasync.DeleteTaskOutput, error)
	ListTasks(ctx context.Context, params *datasync.ListTasksInput, optFns ...func(*datasync.Options)) (*datasync.ListTasksOutput, error)
}

// NewDataSyncTask creates a new DataSync Task resource using the generic resource pattern.
func NewDataSyncTask() AwsResource {
	return NewAwsResource(&resource.Resource[DataSyncTaskAPI]{
		ResourceTypeName: "data-sync-task",
		BatchSize:        19,
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
			if cfg.ShouldInclude(config.ResourceValue{
				Name: task.Name,
			}) {
				identifiers = append(identifiers, task.TaskArn)
			}
		}
	}

	return identifiers, nil
}

// deleteDataSyncTask deletes a single DataSync task.
func deleteDataSyncTask(ctx context.Context, client DataSyncTaskAPI, taskArn *string) error {
	_, err := client.DeleteTask(ctx, &datasync.DeleteTaskInput{
		TaskArn: taskArn,
	})
	return err
}
