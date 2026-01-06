package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

const (
	// cloudfrontDistributionDeployWaitTimeout is the maximum time to wait for a distribution
	// to be deployed after disabling it.
	cloudfrontDistributionDeployWaitTimeout = 30 * time.Minute
)

// CloudfrontDistributionAPI defines the interface for CloudFront distribution operations.
type CloudfrontDistributionAPI interface {
	cloudfront.ListDistributionsAPIClient
	cloudfront.GetDistributionAPIClient
	GetDistributionConfig(ctx context.Context, params *cloudfront.GetDistributionConfigInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetDistributionConfigOutput, error)
	UpdateDistribution(ctx context.Context, params *cloudfront.UpdateDistributionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.UpdateDistributionOutput, error)
	DeleteDistribution(ctx context.Context, params *cloudfront.DeleteDistributionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.DeleteDistributionOutput, error)
}

// NewCloudfrontDistributions creates a new CloudFront distribution resource.
func NewCloudfrontDistributions() AwsResource {
	return NewAwsResource(&resource.Resource[CloudfrontDistributionAPI]{
		ResourceTypeName: "cloudfront-distribution",
		BatchSize:        DefaultBatchSize,
		IsGlobal:         true,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[CloudfrontDistributionAPI], cfg aws.Config) {
			r.Scope.Region = "global"
			r.Client = cloudfront.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.CloudfrontDistribution
		},
		Lister: listCloudfrontDistributions,
		Nuker:  resource.SequentialDeleter(nukeCloudfrontDistribution),
	})
}

// listCloudfrontDistributions retrieves all CloudFront distributions that match the config filters.
func listCloudfrontDistributions(ctx context.Context, client CloudfrontDistributionAPI, _ resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var ids []*string
	paginator := cloudfront.NewListDistributionsPaginator(client, &cloudfront.ListDistributionsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		if page.DistributionList == nil || page.DistributionList.Items == nil {
			continue
		}

		for _, item := range page.DistributionList.Items {
			if cfg.ShouldInclude(config.ResourceValue{Name: item.Id}) {
				ids = append(ids, item.Id)
			}
		}
	}

	return ids, nil
}

// nukeCloudfrontDistribution deletes a single CloudFront distribution.
// CloudFront distributions must be disabled before they can be deleted.
// This function handles the full lifecycle: get config -> disable -> wait -> delete.
func nukeCloudfrontDistribution(ctx context.Context, client CloudfrontDistributionAPI, id *string) error {
	// Step 1: Get the current distribution configuration
	getOutput, err := client.GetDistributionConfig(ctx, &cloudfront.GetDistributionConfigInput{Id: id})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	etag := getOutput.ETag

	// Step 2: Disable the distribution if it's currently enabled
	if aws.ToBool(getOutput.DistributionConfig.Enabled) {
		logging.Debugf("Disabling CloudFront distribution %s", aws.ToString(id))

		getOutput.DistributionConfig.Enabled = aws.Bool(false)
		_, err := client.UpdateDistribution(ctx, &cloudfront.UpdateDistributionInput{
			Id:                 id,
			IfMatch:            etag,
			DistributionConfig: getOutput.DistributionConfig,
		})
		if err != nil {
			return errors.WithStackTrace(err)
		}

		// Step 3: Wait for the distribution to be deployed in disabled state
		logging.Debugf("Waiting for CloudFront distribution %s to be deployed", aws.ToString(id))
		waiter := cloudfront.NewDistributionDeployedWaiter(client)
		if err := waiter.Wait(ctx, &cloudfront.GetDistributionInput{Id: id}, cloudfrontDistributionDeployWaitTimeout); err != nil {
			return errors.WithStackTrace(err)
		}

		// Get the latest ETag after waiting (distribution state may have changed)
		getOutput, err = client.GetDistributionConfig(ctx, &cloudfront.GetDistributionConfigInput{Id: id})
		if err != nil {
			return errors.WithStackTrace(err)
		}
		etag = getOutput.ETag
	}

	// Step 4: Delete the distribution
	logging.Debugf("Deleting CloudFront distribution %s", aws.ToString(id))
	_, err = client.DeleteDistribution(ctx, &cloudfront.DeleteDistributionInput{
		Id:      id,
		IfMatch: etag,
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
