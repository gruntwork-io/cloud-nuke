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

type DataSyncLocation struct {
	BaseAwsResource
	Client            datasynciface.DataSyncAPI
	Region            string
	DataSyncLocations []string
}

func (dsl *DataSyncLocation) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.DataSyncLocation
}

func (dsl *DataSyncLocation) Init(session *session.Session) {
	dsl.Client = datasync.New(session)
}

func (dsl *DataSyncLocation) ResourceName() string { return "data-sync-location" }

func (dsl *DataSyncLocation) ResourceIdentifiers() []string { return dsl.DataSyncLocations }

func (dsl *DataSyncLocation) MaxBatchSize() int { return 19 }

func (dsl *DataSyncLocation) Nuke(identifiers []string) error {
	if err := dsl.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (dsl *DataSyncLocation) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := dsl.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	dsl.DataSyncLocations = aws.StringValueSlice(identifiers)
	return dsl.DataSyncLocations, nil
}
