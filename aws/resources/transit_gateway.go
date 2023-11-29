package resources

import (
	"context"

	"github.com/andrewderr/cloud-nuke-a1/config"
	"github.com/andrewderr/cloud-nuke-a1/logging"
	"github.com/andrewderr/cloud-nuke-a1/report"
	"github.com/andrewderr/cloud-nuke-a1/telemetry"
	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/go-commons/errors"
	commonTelemetry "github.com/gruntwork-io/go-commons/telemetry"
)

// Returns a formatted string of TransitGateway IDs
func (tgw *TransitGateways) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	result, err := tgw.Client.DescribeTransitGateways(&ec2.DescribeTransitGatewaysInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var ids []*string
	for _, transitGateway := range result.TransitGateways {
		if configObj.TransitGateway.ShouldInclude(config.ResourceValue{Time: transitGateway.CreationTime}) &&
			awsgo.StringValue(transitGateway.State) != "deleted" && awsgo.StringValue(transitGateway.State) != "deleting" {
			ids = append(ids, transitGateway.TransitGatewayId)
		}
	}

	return ids, nil
}

// Delete all TransitGateways
func (tgw *TransitGateways) nukeAll(ids []*string) error {
	if len(ids) == 0 {
		logging.Debugf("No Transit Gateways to nuke in region %s", tgw.Region)
		return nil
	}

	logging.Debugf("Deleting all Transit Gateways in region %s", tgw.Region)
	var deletedIds []*string

	for _, id := range ids {
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
			telemetry.TrackEvent(commonTelemetry.EventContext{
				EventName: "Error Nuking Transit Gateway Instance",
			}, map[string]interface{}{
				"region": tgw.Region,
			})
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
