package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/gruntwork-io/aws-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Returns a formatted string of ELBv2 Arns
func getAllElbv2Instances(session *session.Session, region string, excludeSince time.Time) ([]*string, error) {
	svc := elbv2.New(session)
	result, err := svc.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var arns []*string
	for _, balancer := range result.LoadBalancers {
		if excludeSince.Before(*balancer.CreatedTime) {
			arns = append(arns, balancer.LoadBalancerArn)
		}
	}

	return arns, nil
}

// Deletes all Elastic Load Balancers
func nukeAllElbv2Instances(session *session.Session, arns []*string) error {
	svc := elbv2.New(session)

	if len(arns) == 0 {
		logging.Logger.Infof("No V2 Elastic Load Balancers to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all V2 Elastic Load Balancers in region %s", *session.Config.Region)

	for _, arn := range arns {
		params := &elbv2.DeleteLoadBalancerInput{
			LoadBalancerArn: arn,
		}

		_, err := svc.DeleteLoadBalancer(params)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}

		logging.Logger.Infof("Deleted ELBv2: %s", *arn)
	}

	err := svc.WaitUntilLoadBalancersDeleted(&elbv2.DescribeLoadBalancersInput{
		LoadBalancerArns: arns,
	})

	if err != nil {
		return errors.WithStackTrace(err)
	}

	logging.Logger.Infof("[OK] %d V2 Elastic Load Balancer(s) deleted in %s", len(arns), *session.Config.Region)
	return nil
}
