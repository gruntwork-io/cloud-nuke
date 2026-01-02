package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/datasync"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// DataSyncLocationAPI defines the interface for DataSync Location operations.
type DataSyncLocationAPI interface {
	DeleteLocation(ctx context.Context, params *datasync.DeleteLocationInput, optFns ...func(*datasync.Options)) (*datasync.DeleteLocationOutput, error)
	ListLocations(ctx context.Context, params *datasync.ListLocationsInput, optFns ...func(*datasync.Options)) (*datasync.ListLocationsOutput, error)
}

// NewDataSyncLocation creates a new DataSync Location resource using the generic resource pattern.
func NewDataSyncLocation() AwsResource {
	return NewAwsResource(&resource.Resource[DataSyncLocationAPI]{
		ResourceTypeName: "data-sync-location",
		BatchSize:        19,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[DataSyncLocationAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = datasync.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.DataSyncLocation
		},
		Lister: listDataSyncLocations,
		Nuker:  resource.SimpleBatchDeleter(deleteDataSyncLocation),
	})
}

// listDataSyncLocations retrieves all DataSync locations that match the config filters.
// Note: The ListLocations API returns only LocationArn and LocationUri. It does not include
// Name, CreationTime, or Tags. We use LocationUri as the name for filtering purposes since
// it contains descriptive information about the location (e.g., "s3://bucket-name/prefix").
func listDataSyncLocations(ctx context.Context, client DataSyncLocationAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var identifiers []*string

	paginator := datasync.NewListLocationsPaginator(client, &datasync.ListLocationsInput{
		MaxResults: aws.Int32(100),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, location := range page.Locations {
			// Use LocationUri as the name for filtering since LocationListEntry
			// does not include a Name field or CreationTime.
			if cfg.ShouldInclude(config.ResourceValue{
				Name: location.LocationUri,
			}) {
				identifiers = append(identifiers, location.LocationArn)
			}
		}
	}

	return identifiers, nil
}

// deleteDataSyncLocation deletes a single DataSync location.
func deleteDataSyncLocation(ctx context.Context, client DataSyncLocationAPI, locationArn *string) error {
	_, err := client.DeleteLocation(ctx, &datasync.DeleteLocationInput{
		LocationArn: locationArn,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
