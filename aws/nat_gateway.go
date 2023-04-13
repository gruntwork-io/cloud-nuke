package aws

import (
	"fmt"
	"sync"
	"time"

	"github.com/gruntwork-io/cloud-nuke/telemetry"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/gruntwork-io/go-commons/retry"
	multierror "github.com/hashicorp/go-multierror"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
)

func getAllNatGateways(session *session.Session, excludeAfter time.Time, configObj config.Config) ([]*string, error) {
	svc := ec2.New(session)

	allNatGateways := []*string{}
	input := &ec2.DescribeNatGatewaysInput{}
	err := svc.DescribeNatGatewaysPages(
		input,
		func(page *ec2.DescribeNatGatewaysOutput, lastPage bool) bool {
			for _, ngw := range page.NatGateways {
				if shouldIncludeNatGateway(ngw, excludeAfter, configObj) {
					allNatGateways = append(allNatGateways, ngw.NatGatewayId)
				}
			}
			return !lastPage
		},
	)

	return allNatGateways, errors.WithStackTrace(err)
}

func shouldIncludeNatGateway(ngw *ec2.NatGateway, excludeAfter time.Time, configObj config.Config) bool {
	if ngw == nil {
		return false
	}

	ngwState := aws.StringValue(ngw.State)
	if ngwState == ec2.NatGatewayStateDeleted || ngwState == ec2.NatGatewayStateDeleting {
		return false
	}

	if ngw.CreateTime != nil && excludeAfter.Before(*ngw.CreateTime) {
		return false
	}

	if hasNGWExcludeTag(ngw) {
		return false
	}

	if aws.StringValue(ngw.State) == ec2.NatGatewayStateDeleted {
		return false
	}

	return config.ShouldInclude(
		getNatGatewayName(ngw),
		configObj.NatGateway.IncludeRule.NamesRegExp,
		configObj.NatGateway.ExcludeRule.NamesRegExp,
	)
}

// hasNGWExcludeTag checks whether the exlude tag is set for a resource to skip deleting it.
func hasNGWExcludeTag(ngw *ec2.NatGateway) bool {
	// Exclude deletion of any buckets with cloud-nuke-excluded tags
	for _, tag := range ngw.Tags {
		if *tag.Key == AwsResourceExclusionTagKey && *tag.Value == "true" {
			return true
		}
	}
	return false
}

func getNatGatewayName(ngw *ec2.NatGateway) string {
	for _, tag := range ngw.Tags {
		if aws.StringValue(tag.Key) == "Name" {
			return aws.StringValue(tag.Value)
		}
	}
	return ""
}

func nukeAllNatGateways(session *session.Session, identifiers []*string) error {
	region := aws.StringValue(session.Config.Region)

	svc := ec2.New(session)

	if len(identifiers) == 0 {
		logging.Logger.Debugf("No Nat Gateways to nuke in region %s", region)
		return nil
	}

	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on NatGateways.MaxBatchSize, however we add a guard here to warn users when the batching fails and has a
	// chance of throttling AWS. Since we concurrently make one call for each identifier, we pick 100 for the limit here
	// because many APIs in AWS have a limit of 100 requests per second.
	if len(identifiers) > 100 {
		logging.Logger.Debugf("Nuking too many NAT gateways at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyNatErr{}
	}

	// There is no bulk delete nat gateway API, so we delete the batch of nat gateways concurrently using go routines.
	logging.Logger.Debugf("Deleting Nat Gateways in region %s", region)
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, ngwID := range identifiers {
		errChans[i] = make(chan error, 1)
		go deleteNatGatewayAsync(wg, errChans[i], svc, ngwID)
	}
	wg.Wait()

	// Collect all the errors from the async delete calls into a single error struct.
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Logger.Debugf("[Failed] %s", err)
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking NAT Gateway",
			}, map[string]interface{}{
				"region": *session.Config.Region,
			})
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}

	// Now wait until the NAT gateways are deleted
	err := retry.DoWithRetry(
		logging.Logger,
		"Waiting for all NAT gateways to be deleted.",
		// Wait a maximum of 5 minutes: 10 seconds in between, up to 30 times
		30, 10*time.Second,
		func() error {
			areDeleted, err := areAllNatGatewaysDeleted(svc, identifiers)
			if err != nil {
				return errors.WithStackTrace(retry.FatalError{Underlying: err})
			}
			if areDeleted {
				return nil
			}
			return fmt.Errorf("Not all NAT gateways deleted.")
		},
	)
	if err != nil {
		return errors.WithStackTrace(err)
	}
	for _, ngwID := range identifiers {
		logging.Logger.Debugf("[OK] NAT Gateway %s was deleted in %s", aws.StringValue(ngwID), region)
	}
	return nil
}

// areAllNatGatewaysDeleted returns true if all the requested NAT gateways have been deleted. This is determined by
// querying for the statuses of all the NAT gateways, and checking if AWS knows about them (if not, the NAT gateway was
// deleted and rolled off AWS DB) or if the status was updated to deleted.
func areAllNatGatewaysDeleted(svc *ec2.EC2, identifiers []*string) (bool, error) {
	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on NatGateways.MaxBatchSize.
	resp, err := svc.DescribeNatGateways(&ec2.DescribeNatGatewaysInput{NatGatewayIds: identifiers})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NatGatewayNotFound" {
			return true, nil
		}
		return false, err
	}
	if len(resp.NatGateways) == 0 {
		return true, nil
	}
	for _, ngw := range resp.NatGateways {
		if ngw == nil {
			continue
		}

		if aws.StringValue(ngw.State) != ec2.NatGatewayStateDeleted {
			return false, nil
		}
	}
	// At this point, all the NAT gateways are either nil, or in deleted state.
	return true, nil
}

// deleteNatGatewaysAsync deletes the provided NAT Gateway asynchronously in a goroutine, using wait groups for
// concurrency control and a return channel for errors.
func deleteNatGatewayAsync(wg *sync.WaitGroup, errChan chan error, svc *ec2.EC2, ngwID *string) {
	defer wg.Done()

	input := &ec2.DeleteNatGatewayInput{NatGatewayId: ngwID}
	_, err := svc.DeleteNatGateway(input)

	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.StringValue(ngwID),
		ResourceType: "NAT Gateway",
		Error:        err,
	}
	report.Record(e)

	errChan <- err
}

// Custom errors

type TooManyNatErr struct{}

func (err TooManyNatErr) Error() string {
	return "Too many NAT Gateways requested at once."
}
