package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

// LoadBalancersV2PaginatorAPI defines the interface for paginator operations.
type LoadBalancersV2PaginatorAPI interface {
	LoadBalancersV2API
	elasticloadbalancingv2.DescribeLoadBalancersAPIClient
}

// LoadBalancersV2API defines the interface for ELBv2 operations.
type LoadBalancersV2API interface {
	DescribeLoadBalancers(ctx context.Context, params *elasticloadbalancingv2.DescribeLoadBalancersInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error)
	DeleteLoadBalancer(ctx context.Context, params *elasticloadbalancingv2.DeleteLoadBalancerInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DeleteLoadBalancerOutput, error)
}

// NewLoadBalancersV2 creates a new LoadBalancersV2 resource using the generic resource pattern.
func NewLoadBalancersV2() AwsResource {
	return NewAwsResource(&resource.Resource[LoadBalancersV2API]{
		ResourceTypeName: "elbv2",
		BatchSize:        49,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[LoadBalancersV2API], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = elasticloadbalancingv2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ELBv2
		},
		Lister: listLoadBalancersV2,
		Nuker:  resource.SequentialDeleter(resource.DeleteThenWait(deleteLoadBalancerV2, waitForLoadBalancerV2Deleted)),
	})
}

// listLoadBalancersV2 retrieves all ELBv2 load balancers that match the config filters.
func listLoadBalancersV2(ctx context.Context, client LoadBalancersV2API, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	paginator := elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(client.(LoadBalancersV2PaginatorAPI), &elasticloadbalancingv2.DescribeLoadBalancersInput{})

	var arns []*string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, balancer := range page.LoadBalancers {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: balancer.LoadBalancerName,
				Time: balancer.CreatedTime,
			}) {
				arns = append(arns, balancer.LoadBalancerArn)
			}
		}
	}

	return arns, nil
}

// deleteLoadBalancerV2 deletes a single ELBv2 load balancer.
func deleteLoadBalancerV2(ctx context.Context, client LoadBalancersV2API, arn *string) error {
	_, err := client.DeleteLoadBalancer(ctx, &elasticloadbalancingv2.DeleteLoadBalancerInput{
		LoadBalancerArn: arn,
	})
	return err
}

// waitForLoadBalancerV2Deleted waits for an ELBv2 load balancer to be deleted.
func waitForLoadBalancerV2Deleted(ctx context.Context, client LoadBalancersV2API, arn *string) error {
	waiter := elasticloadbalancingv2.NewLoadBalancersDeletedWaiter(client)
	return waiter.Wait(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []string{aws.ToString(arn)},
	}, 5*time.Minute)
}
