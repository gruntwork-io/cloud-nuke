package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
)

// [Note 1] :  NOTE on the Apporach used:-Using the `dry run` approach on verifying the nuking permission in case of a scoped IAM role.
// IAM:simulateCustomPolicy : could also be used but the IAM role itself needs permission for simulateCustomPolicy method
//else this would not get the desired result. Also in case of multiple t-gateway, if only some has permssion to be nuked,
// the t-gateway resource ids needs to be passed individually inside the IAM:simulateCustomPolicy to get the desired result,
// else all would result in `Implicit-deny` as response- this might increase the time complexity.Using dry run to avoid this.

// Returns a formatted string of TransitGateway IDs
func (tgw *TransitGateways) getAll(c context.Context, configObj config.Config) ([]*string, error) {

	result, err := tgw.Client.DescribeTransitGatewaysWithContext(tgw.Context, &ec2.DescribeTransitGatewaysInput{})
	if err != nil {
		logging.Debugf("[DescribeTransitGateways Failed] %s", err)
		return nil, errors.WithStackTrace(err)
	}

	currentOwner := c.Value(util.AccountIdKey)
	var ids []*string
	for _, transitGateway := range result.TransitGateways {
		if configObj.TransitGateway.ShouldInclude(config.ResourceValue{Time: transitGateway.CreationTime}) &&
			awsgo.StringValue(transitGateway.State) != "deleted" && awsgo.StringValue(transitGateway.State) != "deleting" {
			ids = append(ids, transitGateway.TransitGatewayId)
		}

		if currentOwner != nil && transitGateway.OwnerId != nil && currentOwner != awsgo.StringValue(transitGateway.OwnerId) {
			tgw.SetNukableStatus(*transitGateway.TransitGatewayId, util.ErrDifferentOwner)
			continue
		}
	}

	// Check and verfiy the list of allowed nuke actions
	// VerifyNukablePermissions is used to iterate over a list of Transit Gateway IDs (ids) and execute a provided function (func(id *string) error).
	// The function, attempts to delete a Transit Gateway with the specified ID in a dry-run mode (checking permissions without actually performing the delete operation). The result of this operation (error or success) is then captured.
	// See more at [Note 1]
	tgw.VerifyNukablePermissions(ids, func(id *string) error {
		params := &ec2.DeleteTransitGatewayInput{
			TransitGatewayId: id,
			DryRun:           aws.Bool(true), // dry run set as true , checks permission without actualy making the request
		}
		_, err := tgw.Client.DeleteTransitGatewayWithContext(tgw.Context, params)
		return err
	})

	return ids, nil
}

// Delete all TransitGateways
// it attempts to nuke only those resources for which the current IAM user has permission
func (tgw *TransitGateways) nukeAll(ids []*string) error {
	if len(ids) == 0 {
		logging.Debugf("No Transit Gateways to nuke in region %s", tgw.Region)
		return nil
	}

	logging.Debugf("Deleting all Transit Gateways in region %s", tgw.Region)
	var deletedIds []*string

	for _, id := range ids {
		//check the id has the permission to nuke, if not. continue the execution
		if nukable, reason := tgw.IsNukable(*id); !nukable {
			//not adding the report on final result hence not adding a record entry here
			logging.Debugf("[Skipping] %s nuke because %v", *id, reason)
			continue
		}

		params := &ec2.DeleteTransitGatewayInput{
			TransitGatewayId: id,
		}

		_, err := tgw.Client.DeleteTransitGatewayWithContext(tgw.Context, params)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.StringValue(id),
			ResourceType: "Transit Gateway",
			Error:        err,
		}
		report.Record(e)

		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedIds = append(deletedIds, id)
			logging.Debugf("Deleted Transit Gateway: %s", *id)
		}
	}

	logging.Debugf("[OK] %d Transit Gateway(s) deleted in %s", len(deletedIds), tgw.Region)
	return nil
}
