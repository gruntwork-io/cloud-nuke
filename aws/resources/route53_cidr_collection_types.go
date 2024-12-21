package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type Route53CidrCollectionAPI interface {
	ListCidrCollections(ctx context.Context, params *route53.ListCidrCollectionsInput, optFns ...func(*route53.Options)) (*route53.ListCidrCollectionsOutput, error)
	ListCidrBlocks(ctx context.Context, params *route53.ListCidrBlocksInput, optFns ...func(*route53.Options)) (*route53.ListCidrBlocksOutput, error)
	ChangeCidrCollection(ctx context.Context, params *route53.ChangeCidrCollectionInput, optFns ...func(*route53.Options)) (*route53.ChangeCidrCollectionOutput, error)
	DeleteCidrCollection(ctx context.Context, params *route53.DeleteCidrCollectionInput, optFns ...func(*route53.Options)) (*route53.DeleteCidrCollectionOutput, error)
}

// Route53CidrCollection - represents all Route53CidrCollection
type Route53CidrCollection struct {
	BaseAwsResource
	Client Route53CidrCollectionAPI
	Region string
	Ids    []string
}

func (r *Route53CidrCollection) InitV2(cfg aws.Config) {
	r.Client = route53.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (r *Route53CidrCollection) ResourceName() string {
	return "route53-cidr-collection"
}

func (r *Route53CidrCollection) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids of the cidr collections
func (r *Route53CidrCollection) ResourceIdentifiers() []string {
	return r.Ids
}

func (r *Route53CidrCollection) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.Route53CIDRCollection
}

func (r *Route53CidrCollection) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := r.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	r.Ids = aws.ToStringSlice(identifiers)
	return r.Ids, nil
}

// Nuke - nuke 'em all!!!
func (r *Route53CidrCollection) Nuke(identifiers []string) error {
	if err := r.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
