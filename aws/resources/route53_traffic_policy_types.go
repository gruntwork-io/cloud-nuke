package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type Route53TrafficPolicyAPI interface {
	ListTrafficPolicies(ctx context.Context, params *route53.ListTrafficPoliciesInput, optFns ...func(*route53.Options)) (*route53.ListTrafficPoliciesOutput, error)
	DeleteTrafficPolicy(ctx context.Context, params *route53.DeleteTrafficPolicyInput, optFns ...func(*route53.Options)) (*route53.DeleteTrafficPolicyOutput, error)
}

// Route53TrafficPolicy - represents all Route53TrafficPolicy
type Route53TrafficPolicy struct {
	BaseAwsResource
	Client     Route53TrafficPolicyAPI
	Region     string
	Ids        []string
	versionMap map[string]*int32
}

func (r *Route53TrafficPolicy) Init(cfg aws.Config) {
	r.Client = route53.NewFromConfig(cfg)
	r.versionMap = make(map[string]*int32)
}

// ResourceName - the simple name of the aws resource
func (r *Route53TrafficPolicy) ResourceName() string {
	return "route53-traffic-policy"
}

func (r *Route53TrafficPolicy) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids of traffic policies
func (r *Route53TrafficPolicy) ResourceIdentifiers() []string {
	return r.Ids
}
func (r *Route53TrafficPolicy) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.Route53TrafficPolicy
}

func (r *Route53TrafficPolicy) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := r.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	r.Ids = aws.ToStringSlice(identifiers)
	return r.Ids, nil
}

// Nuke - nuke 'em all!!!
func (r *Route53TrafficPolicy) Nuke(ctx context.Context, identifiers []string) error {
	if err := r.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
