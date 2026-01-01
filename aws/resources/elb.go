package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// LoadBalancersAPI defines the interface for ELB operations.
type LoadBalancersAPI interface {
	DescribeLoadBalancers(ctx context.Context, params *elasticloadbalancing.DescribeLoadBalancersInput, optFns ...func(*elasticloadbalancing.Options)) (*elasticloadbalancing.DescribeLoadBalancersOutput, error)
	DeleteLoadBalancer(ctx context.Context, params *elasticloadbalancing.DeleteLoadBalancerInput, optFns ...func(*elasticloadbalancing.Options)) (*elasticloadbalancing.DeleteLoadBalancerOutput, error)
}

// NewLoadBalancers creates a new LoadBalancers resource using the generic resource pattern.
func NewLoadBalancers() AwsResource {
	return NewAwsResource(&resource.Resource[LoadBalancersAPI]{
		ResourceTypeName: "elb",
		BatchSize:        49,
		InitClient: func(r *resource.Resource[LoadBalancersAPI], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for LoadBalancers client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = elasticloadbalancing.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ELBv1
		},
		Lister: listLoadBalancers,
		Nuker:  resource.SequentialDeleter(deleteLoadBalancer),
	})
}

// listLoadBalancers retrieves all Classic ELB load balancers that match the config filters.
func listLoadBalancers(ctx context.Context, client LoadBalancersAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	result, err := client.DescribeLoadBalancers(ctx, &elasticloadbalancing.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string
	for _, balancer := range result.LoadBalancerDescriptions {
		if cfg.ShouldInclude(config.ResourceValue{
			Name: balancer.LoadBalancerName,
			Time: balancer.CreatedTime,
		}) {
			names = append(names, balancer.LoadBalancerName)
		}
	}

	return names, nil
}

// deleteLoadBalancer deletes a single Classic ELB load balancer.
func deleteLoadBalancer(ctx context.Context, client LoadBalancersAPI, name *string) error {
	_, err := client.DeleteLoadBalancer(ctx, &elasticloadbalancing.DeleteLoadBalancerInput{
		LoadBalancerName: name,
	})
	if err != nil {
		return err
	}

	return waitUntilElbDeleted(ctx, client, name)
}

// waitUntilElbDeleted waits until the ELB is deleted.
func waitUntilElbDeleted(ctx context.Context, client LoadBalancersAPI, name *string) error {
	for i := 0; i < 30; i++ {
		output, err := client.DescribeLoadBalancers(ctx, &elasticloadbalancing.DescribeLoadBalancersInput{
			LoadBalancerNames: []string{aws.ToString(name)},
		})
		if err != nil {
			// ELB not found means it's deleted
			return nil
		}

		if len(output.LoadBalancerDescriptions) == 0 {
			return nil
		}

		time.Sleep(1 * time.Second)
		logging.Debug("Waiting for ELB to be deleted")
	}

	return ElbDeleteError{}
}

// ElbDeleteError represents an error when deleting ELB.
type ElbDeleteError struct{}

func (e ElbDeleteError) Error() string {
	return "ELB was not deleted"
}
