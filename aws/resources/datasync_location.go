package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/datasync"
	"github.com/aws/aws-sdk-go-v2/service/datasync/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// DataSyncLocationAPI defines the interface for DataSync Location operations.
type DataSyncLocationAPI interface {
	DeleteLocation(ctx context.Context, params *datasync.DeleteLocationInput, optFns ...func(*datasync.Options)) (*datasync.DeleteLocationOutput, error)
	DescribeLocationAzureBlob(ctx context.Context, params *datasync.DescribeLocationAzureBlobInput, optFns ...func(*datasync.Options)) (*datasync.DescribeLocationAzureBlobOutput, error)
	DescribeLocationEfs(ctx context.Context, params *datasync.DescribeLocationEfsInput, optFns ...func(*datasync.Options)) (*datasync.DescribeLocationEfsOutput, error)
	DescribeLocationFsxLustre(ctx context.Context, params *datasync.DescribeLocationFsxLustreInput, optFns ...func(*datasync.Options)) (*datasync.DescribeLocationFsxLustreOutput, error)
	DescribeLocationFsxOntap(ctx context.Context, params *datasync.DescribeLocationFsxOntapInput, optFns ...func(*datasync.Options)) (*datasync.DescribeLocationFsxOntapOutput, error)
	DescribeLocationFsxOpenZfs(ctx context.Context, params *datasync.DescribeLocationFsxOpenZfsInput, optFns ...func(*datasync.Options)) (*datasync.DescribeLocationFsxOpenZfsOutput, error)
	DescribeLocationFsxWindows(ctx context.Context, params *datasync.DescribeLocationFsxWindowsInput, optFns ...func(*datasync.Options)) (*datasync.DescribeLocationFsxWindowsOutput, error)
	DescribeLocationHdfs(ctx context.Context, params *datasync.DescribeLocationHdfsInput, optFns ...func(*datasync.Options)) (*datasync.DescribeLocationHdfsOutput, error)
	DescribeLocationNfs(ctx context.Context, params *datasync.DescribeLocationNfsInput, optFns ...func(*datasync.Options)) (*datasync.DescribeLocationNfsOutput, error)
	DescribeLocationObjectStorage(ctx context.Context, params *datasync.DescribeLocationObjectStorageInput, optFns ...func(*datasync.Options)) (*datasync.DescribeLocationObjectStorageOutput, error)
	DescribeLocationS3(ctx context.Context, params *datasync.DescribeLocationS3Input, optFns ...func(*datasync.Options)) (*datasync.DescribeLocationS3Output, error)
	DescribeLocationSmb(ctx context.Context, params *datasync.DescribeLocationSmbInput, optFns ...func(*datasync.Options)) (*datasync.DescribeLocationSmbOutput, error)
	ListLocations(ctx context.Context, params *datasync.ListLocationsInput, optFns ...func(*datasync.Options)) (*datasync.ListLocationsOutput, error)
	ListTagsForResource(ctx context.Context, params *datasync.ListTagsForResourceInput, optFns ...func(*datasync.Options)) (*datasync.ListTagsForResourceOutput, error)
}

// NewDataSyncLocation creates a new DataSync Location resource using the generic resource pattern.
func NewDataSyncLocation() AwsResource {
	return NewAwsResource(&resource.Resource[DataSyncLocationAPI]{
		ResourceTypeName: "data-sync-location",
		// DataSync API limit is 20 requests; using 19 to stay safely under the limit.
		BatchSize: 19,
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
			var tags map[string]string
			tagsOutput, err := client.ListTagsForResource(ctx, &datasync.ListTagsForResourceInput{
				ResourceArn: location.LocationArn,
			})
			if err != nil {
				logging.Debugf("Error getting tags for DataSync location %s: %v", aws.ToString(location.LocationArn), err)
				continue
			}
			if tagsOutput != nil {
				tags = util.ConvertDataSyncTagsToMap(tagsOutput.Tags)
			}

			var createdAt *time.Time
			if cfg.HasTimeFilter() && passesNameAndTagRules(cfg, location.LocationUri, tags) {
				createdAt = describeDataSyncLocationCreationTime(ctx, client, location)
			}

			// Use LocationUri as the name for filtering since LocationListEntry
			// does not include a Name field.
			if cfg.ShouldInclude(config.ResourceValue{
				Name: location.LocationUri,
				Time: createdAt,
				Tags: tags,
			}) {
				identifiers = append(identifiers, location.LocationArn)
			}
		}
	}

	return identifiers, nil
}

// describeDataSyncLocationCreationTime reads the creation time that ListLocations
// does not return, with the Describe call for the URI scheme, such as "s3" in
// "s3://bucket/prefix". It warns and returns nil when the call fails.
func describeDataSyncLocationCreationTime(ctx context.Context, client DataSyncLocationAPI, location types.LocationListEntry) *time.Time {
	arn := location.LocationArn
	scheme, _, _ := strings.Cut(aws.ToString(location.LocationUri), "://")

	var createdAt *time.Time
	var err error
	switch scheme {
	case "azure-blob":
		var out *datasync.DescribeLocationAzureBlobOutput
		if out, err = client.DescribeLocationAzureBlob(ctx, &datasync.DescribeLocationAzureBlobInput{LocationArn: arn}); err == nil {
			createdAt = out.CreationTime
		}
	case "efs":
		var out *datasync.DescribeLocationEfsOutput
		if out, err = client.DescribeLocationEfs(ctx, &datasync.DescribeLocationEfsInput{LocationArn: arn}); err == nil {
			createdAt = out.CreationTime
		}
	case "fsxl":
		var out *datasync.DescribeLocationFsxLustreOutput
		if out, err = client.DescribeLocationFsxLustre(ctx, &datasync.DescribeLocationFsxLustreInput{LocationArn: arn}); err == nil {
			createdAt = out.CreationTime
		}
	case "fsxn":
		var out *datasync.DescribeLocationFsxOntapOutput
		if out, err = client.DescribeLocationFsxOntap(ctx, &datasync.DescribeLocationFsxOntapInput{LocationArn: arn}); err == nil {
			createdAt = out.CreationTime
		}
	case "fsxz":
		var out *datasync.DescribeLocationFsxOpenZfsOutput
		if out, err = client.DescribeLocationFsxOpenZfs(ctx, &datasync.DescribeLocationFsxOpenZfsInput{LocationArn: arn}); err == nil {
			createdAt = out.CreationTime
		}
	case "fsxw":
		var out *datasync.DescribeLocationFsxWindowsOutput
		if out, err = client.DescribeLocationFsxWindows(ctx, &datasync.DescribeLocationFsxWindowsInput{LocationArn: arn}); err == nil {
			createdAt = out.CreationTime
		}
	case "hdfs":
		var out *datasync.DescribeLocationHdfsOutput
		if out, err = client.DescribeLocationHdfs(ctx, &datasync.DescribeLocationHdfsInput{LocationArn: arn}); err == nil {
			createdAt = out.CreationTime
		}
	case "nfs":
		var out *datasync.DescribeLocationNfsOutput
		if out, err = client.DescribeLocationNfs(ctx, &datasync.DescribeLocationNfsInput{LocationArn: arn}); err == nil {
			createdAt = out.CreationTime
		}
	case "object-storage":
		var out *datasync.DescribeLocationObjectStorageOutput
		if out, err = client.DescribeLocationObjectStorage(ctx, &datasync.DescribeLocationObjectStorageInput{LocationArn: arn}); err == nil {
			createdAt = out.CreationTime
		}
	case "s3":
		var out *datasync.DescribeLocationS3Output
		if out, err = client.DescribeLocationS3(ctx, &datasync.DescribeLocationS3Input{LocationArn: arn}); err == nil {
			createdAt = out.CreationTime
		}
	case "smb":
		var out *datasync.DescribeLocationSmbOutput
		if out, err = client.DescribeLocationSmb(ctx, &datasync.DescribeLocationSmbInput{LocationArn: arn}); err == nil {
			createdAt = out.CreationTime
		}
	default:
		err = fmt.Errorf("unknown location type %q", scheme)
	}

	if err != nil {
		logging.Warnf("Unable to read the creation time of DataSync location %s, so a time filter cannot include it: %v", aws.ToString(arn), err)
	}
	return createdAt
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
