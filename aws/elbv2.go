package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of ELBv2 Arns
func getAllElbv2Instances(session *session.Session, region string, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := elbv2.New(session)
	result, err := svc.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var arns []*string
	for _, balancer := range result.LoadBalancers {
		if shouldIncludeELBv2(balancer, excludeAfter, configObj) {
			arns = append(arns, balancer.LoadBalancerArn)
		}
	}

	return arns, nil
}

func shouldIncludeELBv2(balancer *elbv2.LoadBalancer, excludeAfter time.Time, configObj config.Config) bool {
	if balancer == nil {
		return false
	}

	if balancer.CreatedTime != nil && excludeAfter.Before(*balancer.CreatedTime) {
		return false
	}

	return config.ShouldInclude(
		awsgo.StringValue(balancer.LoadBalancerName),
		configObj.ELBv2.IncludeRule.NamesRegExp,
		configObj.ELBv2.ExcludeRule.NamesRegExp,
	)
}

// Deletes all Elastic Load Balancers
func nukeAllElbv2Instances(session *session.Session, arns []*string) error {
	svc := elbv2.New(session)

	if len(arns) == 0 {
		logging.Logger.Infof("No V2 Elastic Load Balancers to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all V2 Elastic Load Balancers in region %s", *session.Config.Region)
	var deletedArns []*string

	for _, arn := range arns {
		params := &elbv2.DeleteLoadBalancerInput{
			LoadBalancerArn: arn,
		}

		_, err := svc.DeleteLoadBalancer(params)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			deletedArns = append(deletedArns, arn)
			logging.Logger.Infof("Deleted ELBv2: %s", *arn)
		}
	}

	if len(deletedArns) > 0 {
		err := svc.WaitUntilLoadBalancersDeleted(&elbv2.DescribeLoadBalancersInput{
			LoadBalancerArns: deletedArns,
		})

		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
	}

	logging.Logger.Infof("[OK] %d V2 Elastic Load Balancer(s) deleted in %s", len(deletedArns), *session.Config.Region)
	return nil
}
