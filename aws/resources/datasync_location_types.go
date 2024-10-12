package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/datasync"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type DataSyncLocationAPI interface {
	DeleteLocation(ctx context.Context, params *datasync.DeleteLocationInput, optFns ...func(*datasync.Options)) (*datasync.DeleteLocationOutput, error)
	ListLocations(ctx context.Context, params *datasync.ListLocationsInput, optFns ...func(*datasync.Options)) (*datasync.ListLocationsOutput, error)
}

type DataSyncLocation struct {
	BaseAwsResource
	Client            DataSyncLocationAPI
	Region            string
	DataSyncLocations []string
}

func (dsl *DataSyncLocation) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.DataSyncLocation
}

func (dsl *DataSyncLocation) InitV2(cfg aws.Config) {
	dsl.Client = datasync.NewFromConfig(cfg)
}

func (dsl *DataSyncLocation) IsUsingV2() bool { return true }

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

	dsl.DataSyncLocations = aws.ToStringSlice(identifiers)
	return dsl.DataSyncLocations, nil
}
