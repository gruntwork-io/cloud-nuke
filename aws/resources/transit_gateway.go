package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	goerror "github.com/go-errors/errors"
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

	result, err := tgw.Client.DescribeTransitGateways(&ec2.DescribeTransitGatewaysInput{})
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
		_, err := tgw.Client.DeleteTransitGateway(params)
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
		if nukable, err := tgw.IsNukable(*id); !nukable {
			//not adding the report on final result hence not adding a record entry here
			logging.Debugf("[Skipping] %s nuke because %v", *id, err)
			continue
		}

		params := &ec2.DeleteTransitGatewayInput{
			TransitGatewayId: id,
		}

		_, err := tgw.Client.DeleteTransitGateway(params)

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

// Returns a formatted string of TranstGatewayRouteTable IDs
func (tgw *TransitGatewaysRouteTables) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	// Remove defalt route table, that will be deleted along with its TransitGateway
	param := &ec2.DescribeTransitGatewayRouteTablesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("default-association-route-table"),
				Values: []*string{
					aws.String("false"),
				},
			},
		},
	}

	result, err := tgw.Client.DescribeTransitGatewayRouteTables(param)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var ids []*string
	for _, transitGatewayRouteTable := range result.TransitGatewayRouteTables {
		if configObj.TransitGatewayRouteTable.ShouldInclude(config.ResourceValue{Time: transitGatewayRouteTable.CreationTime}) &&
			awsgo.StringValue(transitGatewayRouteTable.State) != "deleted" && awsgo.StringValue(transitGatewayRouteTable.State) != "deleting" {
			ids = append(ids, transitGatewayRouteTable.TransitGatewayRouteTableId)
		}
	}

	return ids, nil
}

// Delete all TransitGatewayRouteTables
func (tgw *TransitGatewaysRouteTables) nukeAll(ids []*string) error {
	if len(ids) == 0 {
		logging.Debugf("No Transit Gateway Route Tables to nuke in region %s", tgw.Region)
		return nil
	}

	logging.Debugf("Deleting all Transit Gateway Route Tables in region %s", tgw.Region)
	var deletedIds []*string

	for _, id := range ids {
		param := &ec2.DeleteTransitGatewayRouteTableInput{
			TransitGatewayRouteTableId: id,
		}

		_, err := tgw.Client.DeleteTransitGatewayRouteTable(param)
		if err != nil {
			logging.Debugf("[Failed] %s", err)
		} else {
			deletedIds = append(deletedIds, id)
			logging.Debugf("Deleted Transit Gateway Route Table: %s", *id)
		}
	}

	logging.Debugf("[OK] %d Transit Gateway Route Table(s) deleted in %s", len(deletedIds), tgw.Region)
	return nil
}

// Returns a formated string of TransitGatewayVpcAttachment IDs
func (tgw *TransitGatewaysVpcAttachment) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := tgw.Client.DescribeTransitGatewayVpcAttachments(&ec2.DescribeTransitGatewayVpcAttachmentsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var ids []*string
	for _, tgwVpcAttachment := range result.TransitGatewayVpcAttachments {
		if configObj.TransitGatewaysVpcAttachment.ShouldInclude(config.ResourceValue{Time: tgwVpcAttachment.CreationTime}) &&
			awsgo.StringValue(tgwVpcAttachment.State) != "deleted" && awsgo.StringValue(tgwVpcAttachment.State) != "deleting" {
			ids = append(ids, tgwVpcAttachment.TransitGatewayAttachmentId)
		}
	}

	return ids, nil
}

// Delete all TransitGatewayVpcAttachments
func (tgw *TransitGatewaysVpcAttachment) nukeAll(ids []*string) error {
	if len(ids) == 0 {
		logging.Debugf("No Transit Gateway Vpc Attachments to nuke in region %s", tgw.Region)
		return nil
	}

	logging.Debugf("Deleting all Transit Gateway Vpc Attachments in region %s", tgw.Region)
	var deletedIds []*string

	for _, id := range ids {
		param := &ec2.DeleteTransitGatewayVpcAttachmentInput{
			TransitGatewayAttachmentId: id,
		}

		_, err := tgw.Client.DeleteTransitGatewayVpcAttachment(param)

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
			logging.Debugf("Deleted Transit Gateway Vpc Attachment: %s", *id)
		}
	}

	if waiterr := waitForTransitGatewayAttachementToBeDeleted(*tgw); waiterr != nil {
		return errors.WithStackTrace(waiterr)
	}
	logging.Debugf(("[OK] %d Transit Gateway Vpc Attachment(s) deleted in %s"), len(deletedIds), tgw.Region)
	return nil
}

func (tgpa *TransitGatewayPeeringAttachment) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var ids []*string
	err := tgpa.Client.DescribeTransitGatewayPeeringAttachmentsPages(&ec2.DescribeTransitGatewayPeeringAttachmentsInput{}, func(result *ec2.DescribeTransitGatewayPeeringAttachmentsOutput, lastPage bool) bool {
		for _, attachment := range result.TransitGatewayPeeringAttachments {
			if configObj.TransitGatewayPeeringAttachment.ShouldInclude(config.ResourceValue{
				Time: attachment.CreationTime,
			}) {
				ids = append(ids, attachment.TransitGatewayAttachmentId)
			}
		}

		return !lastPage
	})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	return ids, nil
}

func (tgpa *TransitGatewayPeeringAttachment) nukeAll(ids []*string) error {
	for _, id := range ids {
		_, err := tgpa.Client.DeleteTransitGatewayPeeringAttachment(&ec2.DeleteTransitGatewayPeeringAttachmentInput{
			TransitGatewayAttachmentId: id,
		})
		// Record status of this resource
		report.Record(report.Entry{
			Identifier:   aws.StringValue(id),
			ResourceType: tgpa.ResourceName(),
			Error:        err,
		})
		if err != nil {
			logging.Errorf("[Failed] %s", err)
		} else {
			logging.Debugf("Deleted Transit Gateway Peering Attachment: %s", *id)
		}
	}

	return nil
}

func waitForTransitGatewayAttachementToBeDeleted(tgw TransitGatewaysVpcAttachment) error {
	for i := 0; i < 30; i++ {
		gateways, err := tgw.Client.DescribeTransitGatewayVpcAttachments(
			&ec2.DescribeTransitGatewayVpcAttachmentsInput{
				TransitGatewayAttachmentIds: aws.StringSlice(tgw.Ids),
				Filters: []*ec2.Filter{
					{
						Name:   awsgo.String("state"),
						Values: []*string{awsgo.String("deleting")},
					},
				},
			},
		)
		if err != nil {
			return err
		}
		if len(gateways.TransitGatewayVpcAttachments) == 0 {
			return nil
		}
		logging.Info("Waiting for transit gateways attachemensts to be deleted...")
		time.Sleep(10 * time.Second)
	}

	return goerror.New("timed out waiting for transit gateway attahcments to be successfully deleted")
}
