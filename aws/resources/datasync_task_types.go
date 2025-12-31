package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/datasync"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type DataSyncTaskAPI interface {
	DeleteTask(ctx context.Context, params *datasync.DeleteTaskInput, optFns ...func(*datasync.Options)) (*datasync.DeleteTaskOutput, error)
	ListTasks(ctx context.Context, params *datasync.ListTasksInput, optFns ...func(*datasync.Options)) (*datasync.ListTasksOutput, error)
}

type DataSyncTask struct {
	BaseAwsResource
	Client        DataSyncTaskAPI
	Region        string
	DataSyncTasks []string
}

func (dst *DataSyncTask) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.DataSyncTask
}

func (dst *DataSyncTask) Init(cfg aws.Config) {
	dst.Client = datasync.NewFromConfig(cfg)
}

func (dst *DataSyncTask) ResourceName() string { return "data-sync-task" }

func (dst *DataSyncTask) ResourceIdentifiers() []string { return dst.DataSyncTasks }

func (dst *DataSyncTask) MaxBatchSize() int { return 19 }

func (dst *DataSyncTask) Nuke(ctx context.Context, identifiers []string) error {
	if err := dst.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (dst *DataSyncTask) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := dst.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	dst.DataSyncTasks = aws.ToStringSlice(identifiers)
	return dst.DataSyncTasks, nil
}
