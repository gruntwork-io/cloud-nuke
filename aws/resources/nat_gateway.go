package resources

import (
	"context"
	goerror "errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/gruntwork-io/go-commons/retry"
	"github.com/hashicorp/go-multierror"
)

func (ngw *NatGateways) getAll(ctx context.Context, configObj config.Config) ([]*string, error) {
	var allNatGateways []*string

	// Use the paginator for DescribeNatGateways
	paginator := ec2.NewDescribeNatGatewaysPaginator(ngw.Client, &ec2.DescribeNatGatewaysInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, gateway := range page.NatGateways {
			if shouldIncludeNatGateway(gateway, configObj) {
				allNatGateways = append(allNatGateways, gateway.NatGatewayId)
			}
		}
	}

	// checking the nukable permissions
	ngw.VerifyNukablePermissions(allNatGateways, func(id *string) error {
		_, err := ngw.Client.DeleteNatGateway(ctx, &ec2.DeleteNatGatewayInput{
			NatGatewayId: id,
			DryRun:       aws.Bool(true),
		})
		return err
	})

	return allNatGateways, nil
}

func shouldIncludeNatGateway(ngw types.NatGateway, configObj config.Config) bool {

	if ngw.State == types.NatGatewayStateDeleted || ngw.State == types.NatGatewayStateDeleting {
		return false
	}

	return configObj.NatGateway.ShouldInclude(config.ResourceValue{
		Time: ngw.CreateTime,
		Name: getNatGatewayName(ngw),
		Tags: util.ConvertTypesTagsToMap(ngw.Tags),
	})
}

func getNatGatewayName(ngw types.NatGateway) *string {
	for _, tag := range ngw.Tags {
		if aws.ToString(tag.Key) == "Name" {
			return tag.Value
		}
	}

	return nil
}

func (ngw *NatGateways) nukeAll(identifiers []*string) error {
	if len(identifiers) == 0 {
		logging.Debugf("No Nat Gateways to nuke in region %s", ngw.Region)
		return nil
	}

	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on NatGateways.MaxBatchSize, however we add a guard here to warn users when the batching fails and has a
	// chance of throttling AWS. Since we concurrently make one call for each identifier, we pick 100 for the limit here
	// because many APIs in AWS have a limit of 100 requests per second.
	if len(identifiers) > 100 {
		logging.Debugf("Nuking too many NAT gateways at once (100): halting to avoid hitting AWS API rate limiting")
		return TooManyNatErr{}
	}

	// There is no bulk delete nat gateway API, so we delete the batch of nat gateways concurrently using go routines.
	logging.Debugf("Deleting Nat Gateways in region %s", ngw.Region)
	wg := new(sync.WaitGroup)
	wg.Add(len(identifiers))
	errChans := make([]chan error, len(identifiers))
	for i, ngwID := range identifiers {
		errChans[i] = make(chan error, 1)
		go ngw.deleteAsync(wg, errChans[i], ngwID)
	}
	wg.Wait()

	// Collect all the errors from the async delete calls into a single error struct.
	var allErrs *multierror.Error
	for _, errChan := range errChans {
		if err := <-errChan; err != nil {
			allErrs = multierror.Append(allErrs, err)
			logging.Debugf("[Failed] %s", err)
		}
	}
	finalErr := allErrs.ErrorOrNil()
	if finalErr != nil {
		return errors.WithStackTrace(finalErr)
	}

	// Now wait until the NAT gateways are deleted
	err := retry.DoWithRetry(
		logging.Logger.WithTime(time.Now()),
		"Waiting for all NAT gateways to be deleted.",
		// Wait a maximum of 5 minutes: 10 seconds in between, up to 30 times
		30, 10*time.Second,
		func() error {
			areDeleted, err := ngw.areAllNatGatewaysDeleted(identifiers)
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
		logging.Debugf("[OK] NAT Gateway %s was deleted in %s", aws.ToString(ngwID), ngw.Region)
	}
	return nil
}

// areAllNatGatewaysDeleted returns true if all the requested NAT gateways have been deleted. This is determined by
// querying for the statuses of all the NAT gateways, and checking if AWS knows about them (if not, the NAT gateway was
// deleted and rolled off AWS DB) or if the status was updated to deleted.
func (ngw *NatGateways) areAllNatGatewaysDeleted(identifiers []*string) (bool, error) {
	// NOTE: we don't need to do pagination here, because the pagination is handled by the caller to this function,
	// based on NatGateways.MaxBatchSize.
	natGatewayIDs := make([]string, len(identifiers))
	for i, id := range identifiers {
		natGatewayIDs[i] = aws.ToString(id)
	}
	resp, err := ngw.Client.DescribeNatGateways(ngw.Context, &ec2.DescribeNatGatewaysInput{NatGatewayIds: natGatewayIDs})
	if err != nil {
		var apiErr smithy.APIError
		if ok := goerror.As(err, &apiErr); ok && apiErr.ErrorCode() == "NatGatewayNotFound" {
			return true, nil
		}

		return false, err
	}
	if len(resp.NatGateways) == 0 {
		return true, nil
	}
	for _, ngw := range resp.NatGateways {
		if ngw.State != types.NatGatewayStateDeleted {
			return false, nil
		}
	}
	// At this point, all the NAT gateways are either nil, or in deleted state.
	return true, nil
}

// deleteNatGatewaysAsync deletes the provided NAT Gateway asynchronously in a goroutine, using wait groups for
// concurrency control and a return channel for errors.
func (ngw *NatGateways) deleteAsync(wg *sync.WaitGroup, errChan chan error, ngwID *string) {
	defer wg.Done()

	if nukable, reason := ngw.IsNukable(aws.ToString(ngwID)); !nukable {
		logging.Debugf("[Skipping] %s nuke because %v", aws.ToString(ngwID), reason)
		errChan <- nil
		return
	}

	err := nukeNATGateway(ngw.Client, ngwID)
	// Record status of this resource
	e := report.Entry{
		Identifier:   aws.ToString(ngwID),
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

func nukeNATGateway(client NatGatewaysAPI, gateway *string) error {
	logging.Debugf("Deleting NAT gateway %s", aws.ToString(gateway))

	_, err := client.DeleteNatGateway(context.Background(), &ec2.DeleteNatGatewayInput{NatGatewayId: gateway})
	if err != nil {
		logging.Debugf("[Failed] Error deleting NAT gateway %s: %s", aws.ToString(gateway), err)
		return errors.WithStackTrace(err)
	}
	logging.Debugf("[Ok] NAT Gateway deleted successfully %s", aws.ToString(gateway))
	return nil
}
