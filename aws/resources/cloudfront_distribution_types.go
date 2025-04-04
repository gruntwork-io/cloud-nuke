package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type CloudfrontDistributionAPI interface {
	ListDistributions(ctx context.Context, params *cloudfront.ListDistributionsInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListDistributionsOutput, error)
	GetDistributionConfig(ctx context.Context, params *cloudfront.GetDistributionConfigInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetDistributionConfigOutput, error)
	UpdateDistribution(ctx context.Context, params *cloudfront.UpdateDistributionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.UpdateDistributionOutput, error)
	DeleteDistribution(ctx context.Context, params *cloudfront.DeleteDistributionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.DeleteDistributionOutput, error)
}

// CloudfrontDistribution - represents all CloudTrails
type CloudfrontDistribution struct {
	BaseAwsResource
	Client CloudfrontDistributionAPI
	Region string
	Ids    []string
}

func (cd *CloudfrontDistribution) Init(cfg aws.Config) {
	cd.Client = cloudfront.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (cd *CloudfrontDistribution) ResourceName() string {
	return "cloudfront-distribution"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (cd *CloudfrontDistribution) ResourceIdentifiers() []string {
	return cd.Ids
}

func (cd *CloudfrontDistribution) MaxBatchSize() int {
	return 50
}

func (cd *CloudfrontDistribution) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.CloudfrontDistribution
}

func (cd *CloudfrontDistribution) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := cd.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	cd.Ids = aws.ToStringSlice(identifiers)
	return cd.Ids, nil
}

// Nuke - nuke 'em all!!!
func (cd *CloudfrontDistribution) Nuke(identifiers []string) error {
	if err := cd.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
