package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type LoadBalancersV2API interface {
	DescribeLoadBalancers(ctx context.Context, params *elasticloadbalancingv2.DescribeLoadBalancersInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error)
	DeleteLoadBalancer(ctx context.Context, params *elasticloadbalancingv2.DeleteLoadBalancerInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DeleteLoadBalancerOutput, error)
}

// LoadBalancersV2 - represents all load balancers
type LoadBalancersV2 struct {
	BaseAwsResource
	Client LoadBalancersV2API
	Region string
	Arns   []string
}

func (balancer *LoadBalancersV2) InitV2(cfg aws.Config) {
	balancer.Client = elasticloadbalancingv2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (balancer *LoadBalancersV2) ResourceName() string {
	return "elbv2"
}

func (balancer *LoadBalancersV2) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The arns of the load balancers
func (balancer *LoadBalancersV2) ResourceIdentifiers() []string {
	return balancer.Arns
}

func (balancer *LoadBalancersV2) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.ELBv2
}

func (balancer *LoadBalancersV2) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := balancer.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	balancer.Arns = aws.ToStringSlice(identifiers)
	return balancer.Arns, nil
}

// Nuke - nuke 'em all!!!
func (balancer *LoadBalancersV2) Nuke(identifiers []string) error {
	if err := balancer.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
