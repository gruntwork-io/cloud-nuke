package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of ELBv2 Arns
func getAllElbv2Instances(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := elbv2.New(session)
	result, err := svc.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var arns []*string
	for _, balancer := range result.LoadBalancers {
		input := &elbv2.DescribeTagsInput{
			ResourceArns: []*string{
				aws.String(*balancer.LoadBalancerArn),
			},
		}
		tagOutput, err := svc.DescribeTags(input)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}
		if excludeAfter.After(*balancer.CreatedTime) && !hasELBv2ExcludeTag(tagOutput) {
			arns = append(arns, balancer.LoadBalancerArn)
		} else if !hasELBv2ExcludeTag(tagOutput) {
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
