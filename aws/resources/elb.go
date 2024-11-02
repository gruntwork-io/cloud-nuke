package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (balancer *LoadBalancers) waitUntilElbDeleted(input *elasticloadbalancing.DescribeLoadBalancersInput) error {
	for i := 0; i < 30; i++ {
		output, err := balancer.Client.DescribeLoadBalancers(balancer.Context, input)
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

// Returns a formatted string of ELB names
func (balancer *LoadBalancers) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := balancer.Client.DescribeLoadBalancers(balancer.Context, &elasticloadbalancing.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string
	for _, balancer := range result.LoadBalancerDescriptions {
		if configObj.ELBv1.ShouldInclude(config.ResourceValue{
			Name: balancer.LoadBalancerName,
			Time: balancer.CreatedTime,
		}) {
			names = append(names, balancer.LoadBalancerName)
		}
	}

	return names, nil
}

// Deletes all Elastic Load Balancers
func (balancer *LoadBalancers) nukeAll(names []*string) error {
	if len(names) == 0 {
		logging.Debugf("No Elastic Load Balancers to nuke in region %s", balancer.Region)
		return nil
	}

	logging.Debugf("Deleting all Elastic Load Balancers in region %s", balancer.Region)
	var deletedNames []*string

	for _, name := range names {
		params := &elasticloadbalancing.DeleteLoadBalancerInput{
			LoadBalancerName: name,
		}

		_, err := balancer.Client.DeleteLoadBalancer(balancer.Context, params)

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
		err := balancer.waitUntilElbDeleted(&elasticloadbalancing.DescribeLoadBalancersInput{
			LoadBalancerNames: aws.ToStringSlice(deletedNames),
		})
		if err != nil {
			logging.Debugf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
	}

	logging.Debugf("[OK] %d Elastic Load Balancer(s) deleted in %s", len(deletedNames), balancer.Region)
	return nil
}
