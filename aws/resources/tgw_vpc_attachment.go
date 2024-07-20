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
	"github.com/gruntwork-io/go-commons/errors"
)

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
