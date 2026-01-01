package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/go-commons/errors"
)

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
		InitClient: func(r *resource.Resource[LoadBalancersV2API], cfg any) {
			awsCfg, ok := cfg.(aws.Config)
			if !ok {
				logging.Debugf("Invalid config type for LoadBalancersV2 client: expected aws.Config")
				return
			}
			r.Scope.Region = awsCfg.Region
			r.Client = elasticloadbalancingv2.NewFromConfig(awsCfg)
		},
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.ELBv2
		},
		Lister: listLoadBalancersV2,
		Nuker:  deleteLoadBalancersV2,
	})
}

// listLoadBalancersV2 retrieves all ELBv2 load balancers that match the config filters.
func listLoadBalancersV2(ctx context.Context, client LoadBalancersV2API, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	result, err := client.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var arns []*string
	for _, balancer := range result.LoadBalancers {
		if cfg.ShouldInclude(config.ResourceValue{
			Name: balancer.LoadBalancerName,
			Time: balancer.CreatedTime,
		}) {
			arns = append(arns, balancer.LoadBalancerArn)
		}
	}

	return arns, nil
}

// deleteLoadBalancersV2 deletes all ELBv2 load balancers.
func deleteLoadBalancersV2(ctx context.Context, client LoadBalancersV2API, scope resource.Scope, resourceType string, arns []*string) error {
	if len(arns) == 0 {
		logging.Debugf("No V2 Elastic Load Balancers to nuke in region %s", scope.Region)
		return nil
	}

	logging.Debugf("Deleting all V2 Elastic Load Balancers in region %s", scope.Region)
	var deletedArns []*string

	for _, arn := range arns {
		params := &elasticloadbalancingv2.DeleteLoadBalancerInput{
			LoadBalancerArn: arn,
		}

		_, err := client.DeleteLoadBalancer(ctx, params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(arn),
			ResourceType: "Load Balancer (v2)",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedArns = append(deletedArns, arn)
			logging.Debugf("Deleted ELBv2: %s", *arn)
		}
	}

	if len(deletedArns) > 0 {
		waiter := elasticloadbalancingv2.NewLoadBalancersDeletedWaiter(client)
		err := waiter.Wait(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{
			LoadBalancerArns: aws.ToStringSlice(deletedArns),
		}, 5*time.Minute)
		if err != nil {
			logging.Debugf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
	}

	logging.Debugf("[OK] %d V2 Elastic Load Balancer(s) deleted in %s", len(deletedArns), scope.Region)
	return nil
}
