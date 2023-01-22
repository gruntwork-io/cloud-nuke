package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of ELBv2 Arns
func getAllElbv2Instances(session *session.Session, region string, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := elbv2.New(session)
	input := &elbv2.DescribeTagsInput{}
	result, err := svc.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	tags, err := svc.DescribeTags(input)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var arns []*string
	for _, balancer := range result.LoadBalancers {
		if shouldIncludeELBv2(balancer, excludeAfter, configObj, tags) {
			arns = append(arns, balancer.LoadBalancerArn)
		}
	}

	return arns, nil
}

// hasELBv2ExcludeTag checks whether the exlude tag is set for a resource to skip deleting it.
func hasELBv2ExcludeTag(tagOutput *elbv2.DescribeTagsOutput) bool {
	// Exclude deletion of any buckets with cloud-nuke-excluded tags
	for _, tagDescription := range tagOutput.TagDescriptions {
		for _, tag := range tagDescription.Tags {
			if *tag.Key == AwsResourceExclusionTagKey && *tag.Value == "true" {
				return true
			}
		}
	}
	return false
}

func shouldIncludeELBv2(balancer *elbv2.LoadBalancer, excludeAfter time.Time, configObj config.Config, tags *elbv2.DescribeTagsOutput) bool {
	if balancer == nil {
		return false
	}

	if balancer.CreatedTime != nil && excludeAfter.Before(*balancer.CreatedTime) {
		return false
	}

	if hasELBv2ExcludeTag(tags) {
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
		logging.Logger.Debugf("No V2 Elastic Load Balancers to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all V2 Elastic Load Balancers in region %s", *session.Config.Region)
	var deletedArns []*string

	for _, arn := range arns {
		params := &elbv2.DeleteLoadBalancerInput{
			LoadBalancerArn: arn,
		}

		_, err := svc.DeleteLoadBalancer(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(arn),
			ResourceType: "Load Balancer (v2)",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
		} else {
			deletedArns = append(deletedArns, arn)
			logging.Logger.Debugf("Deleted ELBv2: %s", *arn)
		}
	}

	if len(deletedArns) > 0 {
		err := svc.WaitUntilLoadBalancersDeleted(&elbv2.DescribeLoadBalancersInput{
			LoadBalancerArns: deletedArns,
		})
		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
	}

	logging.Logger.Debugf("[OK] %d V2 Elastic Load Balancer(s) deleted in %s", len(deletedArns), *session.Config.Region)
	return nil
}
