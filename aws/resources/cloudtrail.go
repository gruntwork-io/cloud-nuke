package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// CloudtrailTrailAPI defines the interface for CloudTrail operations.
type CloudtrailTrailAPI interface {
	ListTrails(ctx context.Context, params *cloudtrail.ListTrailsInput, optFns ...func(*cloudtrail.Options)) (*cloudtrail.ListTrailsOutput, error)
	DeleteTrail(ctx context.Context, params *cloudtrail.DeleteTrailInput, optFns ...func(*cloudtrail.Options)) (*cloudtrail.DeleteTrailOutput, error)
	ListTags(ctx context.Context, params *cloudtrail.ListTagsInput, optFns ...func(*cloudtrail.Options)) (*cloudtrail.ListTagsOutput, error)
}

// NewCloudtrailTrail creates a new CloudtrailTrail resource using the generic resource pattern.
func NewCloudtrailTrail() AwsResource {
	return NewAwsResource(&resource.Resource[CloudtrailTrailAPI]{
		ResourceTypeName: "cloudtrail",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[CloudtrailTrailAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = cloudtrail.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.CloudtrailTrail
		},
		Lister: listCloudtrailTrails,
		Nuker:  resource.SimpleBatchDeleter(deleteCloudtrailTrail),
	})
}

// listCloudtrailTrails retrieves all CloudTrail trails that match the config filters.
func listCloudtrailTrails(ctx context.Context, client CloudtrailTrailAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var trailArns []*string
	paginator := cloudtrail.NewListTrailsPaginator(client, &cloudtrail.ListTrailsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		// Process trails individually to avoid CloudTrailARNInvalidException when mixing
		// organization trails (from AWS Control Tower) with account-level trails.
		// AWS ListTags API doesn't allow resources from multiple owners in a single call.
		for _, trail := range page.Trails {
			rv := config.ResourceValue{Name: trail.Name, Tags: make(map[string]string)}

			if tags, err := client.ListTags(ctx, &cloudtrail.ListTagsInput{
				ResourceIdList: []string{*trail.TrailARN},
			}); err == nil && len(tags.ResourceTagList) > 0 {
				for _, tag := range tags.ResourceTagList[0].TagsList {
					rv.Tags[*tag.Key] = *tag.Value
				}
			}

			if cfg.ShouldInclude(rv) {
				trailArns = append(trailArns, trail.TrailARN)
			}
		}
	}

	return trailArns, nil
}

// deleteCloudtrailTrail deletes a single CloudTrail trail.
func deleteCloudtrailTrail(ctx context.Context, client CloudtrailTrailAPI, arn *string) error {
	_, err := client.DeleteTrail(ctx, &cloudtrail.DeleteTrailInput{
		Name: arn,
	})
	return err
}
