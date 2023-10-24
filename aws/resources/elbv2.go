package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

// Returns a formatted string of ELBv2 Arns
func (balancer *LoadBalancersV2) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := balancer.Client.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{})
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
		params := &elbv2.DeleteLoadBalancerInput{
			LoadBalancerArn: arn,
		}

		_, err := balancer.Client.DeleteLoadBalancer(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(arn),
			ResourceType: "Load Balancer (v2)",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Load Balancer V2",
			}, map[string]interface{}{
				"region": balancer.Region,
			})
		} else {
			deletedArns = append(deletedArns, arn)
			logging.Debugf("Deleted ELBv2: %s", *arn)
		}
	}

	if len(deletedArns) > 0 {
		err := balancer.Client.WaitUntilLoadBalancersDeleted(&elbv2.DescribeLoadBalancersInput{
			LoadBalancerArns: deletedArns,
		})
		if err != nil {
			logging.Debugf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
	}

	logging.Debugf("[OK] %d V2 Elastic Load Balancer(s) deleted in %s", len(deletedArns), balancer.Region)
	return nil
}
