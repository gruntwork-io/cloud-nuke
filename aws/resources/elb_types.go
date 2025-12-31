package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type LoadBalancersAPI interface {
	DescribeLoadBalancers(ctx context.Context, params *elasticloadbalancing.DescribeLoadBalancersInput, optFns ...func(*elasticloadbalancing.Options)) (*elasticloadbalancing.DescribeLoadBalancersOutput, error)
	DeleteLoadBalancer(ctx context.Context, params *elasticloadbalancing.DeleteLoadBalancerInput, optFns ...func(*elasticloadbalancing.Options)) (*elasticloadbalancing.DeleteLoadBalancerOutput, error)
}

// LoadBalancers - represents all load balancers
type LoadBalancers struct {
	BaseAwsResource
	Client LoadBalancersAPI
	Region string
	Names  []string
}

func (balancer *LoadBalancers) Init(cfg aws.Config) {
	balancer.Client = elasticloadbalancing.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (balancer *LoadBalancers) ResourceName() string {
	return "elb"
}

// ResourceIdentifiers - The names of the load balancers
func (balancer *LoadBalancers) ResourceIdentifiers() []string {
	return balancer.Names
}

func (balancer *LoadBalancers) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (balancer *LoadBalancers) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.ELBv1
}

func (balancer *LoadBalancers) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := balancer.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	balancer.Names = aws.ToStringSlice(identifiers)
	return balancer.Names, nil
}

// Nuke - nuke 'em all!!!
func (balancer *LoadBalancers) Nuke(ctx context.Context, identifiers []string) error {
	if err := balancer.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type ElbDeleteError struct{}

func (e ElbDeleteError) Error() string {
	return "ELB was not deleted"
}
