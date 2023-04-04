package aws

import (
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func waitUntilElbDeleted(svc *elb.ELB, input *elb.DescribeLoadBalancersInput) error {
	for i := 0; i < 30; i++ {
		_, err := svc.DescribeLoadBalancers(input)
		if err != nil {
			if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == "LoadBalancerNotFound" {
				return nil
			}

			return err
		}

		time.Sleep(1 * time.Second)
		logging.Logger.Debug("Waiting for ELB to be deleted")
	}

	return ElbDeleteError{}
}

// Returns a formatted string of ELB names
func getAllElbInstances(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := elb.New(session)
	result, err := svc.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var names []*string
	for _, balancer := range result.LoadBalancerDescriptions {
		if excludeAfter.After(*balancer.CreatedTime) {
			names = append(names, balancer.LoadBalancerName)
		}
	}

	return names, nil
}

// Deletes all Elastic Load Balancers
func nukeAllElbInstances(session *session.Session, names []*string) error {
	svc := elb.New(session)

	if len(names) == 0 {
		logging.Logger.Debugf("No Elastic Load Balancers to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Debugf("Deleting all Elastic Load Balancers in region %s", *session.Config.Region)
	var deletedNames []*string

	for _, name := range names {
		params := &elb.DeleteLoadBalancerInput{
			LoadBalancerName: name,
		}

		_, err := svc.DeleteLoadBalancer(params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(name),
			ResourceType: "Load Balancer (v1)",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Load Balancer (v1)",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
		} else {
			deletedNames = append(deletedNames, name)
			logging.Logger.Debugf("Deleted ELB: %s", *name)
		}
	}

	if len(deletedNames) > 0 {
		err := waitUntilElbDeleted(svc, &elb.DescribeLoadBalancersInput{
			LoadBalancerNames: deletedNames,
		})
		if err != nil {
			logging.Logger.Debugf("[Failed] %s", err)
			return errors.WithStackTrace(err)
		}
	}

	logging.Logger.Debugf("[OK] %d Elastic Load Balancer(s) deleted in %s", len(deletedNames), *session.Config.Region)
	return nil
}
