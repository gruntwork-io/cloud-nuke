package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type Route53HostedZoneAPI interface {
	ListHostedZones(ctx context.Context, params *route53.ListHostedZonesInput, optFns ...func(*route53.Options)) (*route53.ListHostedZonesOutput, error)
	DeleteTrafficPolicyInstance(ctx context.Context, params *route53.DeleteTrafficPolicyInstanceInput, optFns ...func(*route53.Options)) (*route53.DeleteTrafficPolicyInstanceOutput, error)
	DeleteHostedZone(ctx context.Context, params *route53.DeleteHostedZoneInput, optFns ...func(*route53.Options)) (*route53.DeleteHostedZoneOutput, error)
	ListResourceRecordSets(ctx context.Context, params *route53.ListResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error)
	ChangeResourceRecordSets(ctx context.Context, params *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error)
	ListTagsForResources(ctx context.Context, params *route53.ListTagsForResourcesInput, optFns ...func(*route53.Options)) (*route53.ListTagsForResourcesOutput, error)
}

// Route53HostedZone - represents all Route53HostedZone
type Route53HostedZone struct {
	BaseAwsResource
	Client             Route53HostedZoneAPI
	Region             string
	Ids                []string
	HostedZonesDomains map[string]*types.HostedZone
}

func (r *Route53HostedZone) Init(cfg aws.Config) {
	r.Client = route53.NewFromConfig(cfg)
	r.HostedZonesDomains = make(map[string]*types.HostedZone)
}

// ResourceName - the simple name of the aws resource
func (r *Route53HostedZone) ResourceName() string {
	return "route53-hosted-zone"
}

func (r *Route53HostedZone) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids hosted zones
func (r *Route53HostedZone) ResourceIdentifiers() []string {
	return r.Ids
}

func (r *Route53HostedZone) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.Route53HostedZone
}

func (r *Route53HostedZone) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := r.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	r.Ids = aws.ToStringSlice(identifiers)
	return r.Ids, nil
}

// Nuke - nuke 'em all!!!
func (r *Route53HostedZone) Nuke(identifiers []string) error {
	if err := r.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
