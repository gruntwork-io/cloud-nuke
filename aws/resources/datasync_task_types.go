package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/datasync"
	"github.com/aws/aws-sdk-go/service/datasync/datasynciface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type DataSyncTask struct {
	BaseAwsResource
	Client        datasynciface.DataSyncAPI
	Region        string
	DataSyncTasks []string
}

func (dst *DataSyncTask) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.DataSyncTask
}

func (dst *DataSyncTask) Init(session *session.Session) {
	dst.Client = datasync.New(session)
}

func (dst *DataSyncTask) ResourceName() string { return "data-sync-task" }

func (dst *DataSyncTask) ResourceIdentifiers() []string { return dst.DataSyncTasks }

func (dst *DataSyncTask) MaxBatchSize() int { return 19 }

func (dst *DataSyncTask) Nuke(identifiers []string) error {
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

	dst.DataSyncTasks = aws.StringValueSlice(identifiers)
	return dst.DataSyncTasks, nil
}
