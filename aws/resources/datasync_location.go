package resources

import (
	"context"
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

// dataSyncLocationCreationTime reads the creation time of one location type.
type dataSyncLocationCreationTime func(ctx context.Context, client DataSyncLocationAPI, arn *string) (*time.Time, error)

// dataSyncLocationCreationTimes holds one reader per location type, keyed by the
// scheme of the location URI, such as "s3" in "s3://bucket/prefix".
var dataSyncLocationCreationTimes = map[string]dataSyncLocationCreationTime{
	"azure-blob": func(ctx context.Context, client DataSyncLocationAPI, arn *string) (*time.Time, error) {
		out, err := client.DescribeLocationAzureBlob(ctx, &datasync.DescribeLocationAzureBlobInput{LocationArn: arn})
		if err != nil {
			return nil, err
		}
		return out.CreationTime, nil
	},
	"efs": func(ctx context.Context, client DataSyncLocationAPI, arn *string) (*time.Time, error) {
		out, err := client.DescribeLocationEfs(ctx, &datasync.DescribeLocationEfsInput{LocationArn: arn})
		if err != nil {
			return nil, err
		}
		return out.CreationTime, nil
	},
	"fsxl": func(ctx context.Context, client DataSyncLocationAPI, arn *string) (*time.Time, error) {
		out, err := client.DescribeLocationFsxLustre(ctx, &datasync.DescribeLocationFsxLustreInput{LocationArn: arn})
		if err != nil {
			return nil, err
		}
		return out.CreationTime, nil
	},
	"fsxn": func(ctx context.Context, client DataSyncLocationAPI, arn *string) (*time.Time, error) {
		out, err := client.DescribeLocationFsxOntap(ctx, &datasync.DescribeLocationFsxOntapInput{LocationArn: arn})
		if err != nil {
			return nil, err
		}
		return out.CreationTime, nil
	},
	"fsxz": func(ctx context.Context, client DataSyncLocationAPI, arn *string) (*time.Time, error) {
		out, err := client.DescribeLocationFsxOpenZfs(ctx, &datasync.DescribeLocationFsxOpenZfsInput{LocationArn: arn})
		if err != nil {
			return nil, err
		}
		return out.CreationTime, nil
	},
	"fsxw": func(ctx context.Context, client DataSyncLocationAPI, arn *string) (*time.Time, error) {
		out, err := client.DescribeLocationFsxWindows(ctx, &datasync.DescribeLocationFsxWindowsInput{LocationArn: arn})
		if err != nil {
			return nil, err
		}
		return out.CreationTime, nil
	},
	"hdfs": func(ctx context.Context, client DataSyncLocationAPI, arn *string) (*time.Time, error) {
		out, err := client.DescribeLocationHdfs(ctx, &datasync.DescribeLocationHdfsInput{LocationArn: arn})
		if err != nil {
			return nil, err
		}
		return out.CreationTime, nil
	},
	"nfs": func(ctx context.Context, client DataSyncLocationAPI, arn *string) (*time.Time, error) {
		out, err := client.DescribeLocationNfs(ctx, &datasync.DescribeLocationNfsInput{LocationArn: arn})
		if err != nil {
			return nil, err
		}
		return out.CreationTime, nil
	},
	"object-storage": func(ctx context.Context, client DataSyncLocationAPI, arn *string) (*time.Time, error) {
		out, err := client.DescribeLocationObjectStorage(ctx, &datasync.DescribeLocationObjectStorageInput{LocationArn: arn})
		if err != nil {
			return nil, err
		}
		return out.CreationTime, nil
	},
	"s3": func(ctx context.Context, client DataSyncLocationAPI, arn *string) (*time.Time, error) {
		out, err := client.DescribeLocationS3(ctx, &datasync.DescribeLocationS3Input{LocationArn: arn})
		if err != nil {
			return nil, err
		}
		return out.CreationTime, nil
	},
	"smb": func(ctx context.Context, client DataSyncLocationAPI, arn *string) (*time.Time, error) {
		out, err := client.DescribeLocationSmb(ctx, &datasync.DescribeLocationSmbInput{LocationArn: arn})
		if err != nil {
			return nil, err
		}
		return out.CreationTime, nil
	},
}

// describeDataSyncLocationCreationTime reads the creation time of a location,
// which ListLocations does not return. Each location type has its own Describe
// call, chosen by the URI scheme. It returns nil, and warns, when the time
// cannot be read, since a time filter then cannot include the location.
func describeDataSyncLocationCreationTime(ctx context.Context, client DataSyncLocationAPI, location types.LocationListEntry) *time.Time {
	arn := aws.ToString(location.LocationArn)
	scheme, _, _ := strings.Cut(aws.ToString(location.LocationUri), "://")

	read, ok := dataSyncLocationCreationTimes[scheme]
	if !ok {
		logging.Warnf("Unable to read the creation time of DataSync location %s, so a time filter cannot include it: unknown location type %q", arn, scheme)
		return nil
	}

	createdAt, err := read(ctx, client, location.LocationArn)
	if err != nil {
		logging.Warnf("Unable to read the creation time of DataSync location %s, so a time filter cannot include it: %v", arn, err)
		return nil
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
