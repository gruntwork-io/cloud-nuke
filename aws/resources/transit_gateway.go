package resources

import (
	"context"
	"time"
	"fmt"

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

const (
	TransitGatewayAttachmentTypePeering = "peering"
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
		hostNameTagValue := util.GetEC2ResourceNameTagValue(transitGateway.Tags)

		if configObj.TransitGateway.ShouldInclude(config.ResourceValue{
			Time: transitGateway.CreationTime,
			Name: hostNameTagValue,
		}) &&
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

func (tgw *TransitGateways) nuke(id *string) error {
	// check the transit gateway has attachments and nuke them before
	if err := tgw.nukeAttachments(id); err != nil {
		return errors.WithStackTrace(err)
	}

	if _, err := tgw.Client.DeleteTransitGatewayWithContext(tgw.Context, &ec2.DeleteTransitGatewayInput{
		TransitGatewayId: id,
	}); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

func (tgw *TransitGateways) nukeAttachments(id *string) error {
	logging.Debugf("nuking transit gateway attachments for %v", awsgo.StringValue(id))
	output, err := tgw.Client.DescribeTransitGatewayAttachmentsWithContext(tgw.Context,&ec2.DescribeTransitGatewayAttachmentsInput{
		Filters: []*ec2.Filter{
			{
				Name: awsgo.String("transit-gateway-id"),
				Values: []*string{
					id,
				},
			},
			{
				Name: awsgo.String("state"),
				Values: []*string{
					awsgo.String("available"),
				},
			},
		},
	})
	if err != nil {
		logging.Errorf("[Failed] unable to describe the  transit gateway attachments for %v : %s", awsgo.StringValue(id), err)
		return errors.WithStackTrace(err)
	}

	logging.Debugf("%v attachment(s) found with %v", len(output.TransitGatewayAttachments),awsgo.StringValue(id))

	for _, attachments := range output.TransitGatewayAttachments {
		var  (
			err error
			attachmentType = awsgo.StringValue(attachments.ResourceType)
			now = time.Now()
		)

		switch attachmentType {
		case TransitGatewayAttachmentTypePeering:
			logging.Debugf("[Execution] deleting the attachments of type %v for %v ", attachmentType, awsgo.StringValue(id))
			_, err = tgw.Client.DeleteTransitGatewayPeeringAttachmentWithContext(context.Background(), &ec2.DeleteTransitGatewayPeeringAttachmentInput{
				TransitGatewayAttachmentId: attachments.TransitGatewayAttachmentId,
			})
		default:
			err = fmt.Errorf("%v typed transit gateway attachment nuking not handled.", attachmentType)
		}
		if err != nil {
			logging.Errorf("[Failed] unable to delete the  transit gateway peernig attachment for %v : %s", awsgo.StringValue(id), err)
			return err
		}

		err = tgw.WaitUntilTransitGatewayAttachmentDeleted(id, attachmentType)
		if err != nil {
			logging.Errorf("[Failed] unable to wait until nuking the transit gateway attachment with type %v for %v : %s", attachmentType,awsgo.StringValue(id), err)
			return err
		}

		logging.Debugf("waited %v to nuke the attachment", time.Since(now))
	}

	logging.Debugf("[Ok] successfully nuked all the attachments on %v", awsgo.StringValue(id))
	return nil
}

func (tgw *TransitGateways) WaitUntilTransitGatewayAttachmentDeleted(id *string, attachmentType string) error {
	timeoutCtx, cancel := context.WithTimeout(tgw.Context, 5*time.Minute)
	defer cancel()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("transit gateway attachments deletion check timed out after 5 minute")
		case <-ticker.C:
			output, err := tgw.Client.DescribeTransitGatewayAttachmentsWithContext(tgw.Context,&ec2.DescribeTransitGatewayAttachmentsInput{
				Filters: []*ec2.Filter{
					{
						Name: awsgo.String("transit-gateway-id"),
						Values: []*string{
							id,
						},
					},
					{
						Name: awsgo.String("state"),
						Values: []*string{
							awsgo.String("available"),
							awsgo.String("deleting"),
						},
					},
				},
			})
			if err != nil {
				logging.Debugf("transit gateway attachment(s) as type %v existance checking error : %v", attachmentType,err)
				return err
			}

			if len(output.TransitGatewayAttachments) == 0 {
				return nil
			}
			logging.Debugf("%v transit gateway attachments(s) still exists as type %v, waiting...", len(output.TransitGatewayAttachments), attachmentType)
		}
	}
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
		err := tgw.nuke(id)
		
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

	result, err := tgw.Client.DescribeTransitGatewayRouteTablesWithContext(tgw.Context, param)
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

		_, err := tgw.Client.DeleteTransitGatewayRouteTableWithContext(tgw.Context, param)
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
	result, err := tgw.Client.DescribeTransitGatewayVpcAttachmentsWithContext(tgw.Context, &ec2.DescribeTransitGatewayVpcAttachmentsInput{})
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

		_, err := tgw.Client.DeleteTransitGatewayVpcAttachmentWithContext(tgw.Context, param)

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
	err := tgpa.Client.DescribeTransitGatewayPeeringAttachmentsPagesWithContext(tgpa.Context, &ec2.DescribeTransitGatewayPeeringAttachmentsInput{}, func(result *ec2.DescribeTransitGatewayPeeringAttachmentsOutput, lastPage bool) bool {
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
		_, err := tgpa.Client.DeleteTransitGatewayPeeringAttachmentWithContext(tgpa.Context, &ec2.DeleteTransitGatewayPeeringAttachmentInput{
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
