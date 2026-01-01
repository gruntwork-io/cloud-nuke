package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
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
		Nuker:  deleteLoadBalancers,
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

// waitUntilElbDeleted waits until the ELB is deleted.
func waitUntilElbDeleted(ctx context.Context, client LoadBalancersAPI, input *elasticloadbalancing.DescribeLoadBalancersInput) error {
	for i := 0; i < 30; i++ {
		output, err := client.DescribeLoadBalancers(ctx, input)
		if err != nil {
			return err
		}

		if len(output.LoadBalancerDescriptions) == 0 {
			return nil
		}

		time.Sleep(1 * time.Second)
		logging.Debug("Waiting for ELB to be deleted")
	}

	return ElbDeleteError{}
}

// deleteLoadBalancers deletes all Classic ELB load balancers.
func deleteLoadBalancers(ctx context.Context, client LoadBalancersAPI, scope resource.Scope, resourceType string, names []*string) error {
	if len(names) == 0 {
		logging.Debugf("No Elastic Load Balancers to nuke in region %s", scope.Region)
		return nil
	}

	logging.Debugf("Deleting all Elastic Load Balancers in region %s", scope.Region)
	var deletedNames []*string

	for _, name := range names {
		params := &elasticloadbalancing.DeleteLoadBalancerInput{
			LoadBalancerName: name,
		}

		_, err := client.DeleteLoadBalancer(ctx, params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(name),
			ResourceType: "Load Balancer (v1)",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedNames = append(deletedNames, name)
			logging.Debugf("Deleted ELB: %s", *name)
		}
	}

	if len(deletedNames) > 0 {
		err := waitUntilElbDeleted(ctx, client, &elasticloadbalancing.DescribeLoadBalancersInput{
			LoadBalancerNames: aws.ToStringSlice(deletedNames),
		})
		if err != nil {
			logging.Debugf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
	}

	logging.Debugf("[OK] %d Elastic Load Balancer(s) deleted in %s", len(deletedNames), scope.Region)
	return nil
}

// ElbDeleteError represents an error when deleting ELB.
type ElbDeleteError struct{}

func (e ElbDeleteError) Error() string {
	return "ELB was not deleted"
}
