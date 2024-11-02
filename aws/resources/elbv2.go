package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of ELBv2 ARNs
func (balancer *LoadBalancersV2) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := balancer.Client.DescribeLoadBalancers(balancer.Context, &elasticloadbalancingv2.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var arns []*string
	for _, balancer := range result.LoadBalancers {
		if configObj.ELBv2.ShouldInclude(config.ResourceValue{
			Name: balancer.LoadBalancerName,
			Time: balancer.CreatedTime,
		}) {
			arns = append(arns, balancer.LoadBalancerArn)
		}
	}

	return arns, nil
}

// Deletes all Elastic Load Balancers
func (balancer *LoadBalancersV2) nukeAll(arns []*string) error {
	if len(arns) == 0 {
		logging.Debugf("No V2 Elastic Load Balancers to nuke in region %s", balancer.Region)
		return nil
	}

	logging.Debugf("Deleting all V2 Elastic Load Balancers in region %s", balancer.Region)
	var deletedArns []*string

	for _, arn := range arns {
		params := &elasticloadbalancingv2.DeleteLoadBalancerInput{
			LoadBalancerArn: arn,
		}

		_, err := balancer.Client.DeleteLoadBalancer(balancer.Context, params)

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
		waiter := elasticloadbalancingv2.NewLoadBalancersDeletedWaiter(balancer.Client)
		err := waiter.Wait(balancer.Context, &elasticloadbalancingv2.DescribeLoadBalancersInput{
			LoadBalancerArns: aws.ToStringSlice(deletedArns),
		}, 15*time.Minute)
		if err != nil {
			logging.Debugf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
	}

	logging.Debugf("[OK] %d V2 Elastic Load Balancer(s) deleted in %s", len(deletedArns), balancer.Region)
	return nil
}
