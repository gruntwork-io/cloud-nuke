package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/gruntwork-io/aws-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Returns a formatted string of ELB names
func getAllElbInstances(session *session.Session, region string) ([]*string, error) {
	svc := elb.New(session)
	result, err := svc.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string
	for _, balancer := range result.LoadBalancerDescriptions {
		names = append(names, balancer.LoadBalancerName)
	}

	return names, nil
}

// Deletes all Elastic Load Balancers
func nukeAllElbInstances(session *session.Session, names []*string) error {
	svc := elb.New(session)

	if len(names) == 0 {
		logging.Logger.Infof("No Elastic Load Balancers to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all Elastic Load Balancers in region %s", *session.Config.Region)

	for _, name := range names {
		params := &elb.DeleteLoadBalancerInput{
			LoadBalancerName: name,
		}

		_, err := svc.DeleteLoadBalancer(params)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
	}

	logging.Logger.Infof("[OK] %d Elastic Load Balancer(s) deleted in %s", len(names), *session.Config.Region)
	return nil
}
