package resources

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	goerror "github.com/go-errors/errors"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of TransitGatewayVpcAttachment IDs
func (tgw *TransitGatewaysVpcAttachment) getAll(ctx context.Context, configObj config.Config) ([]*string, error) {
	var identifiers []*string
	params := &ec2.DescribeTransitGatewayVpcAttachmentsInput{}

	hasMorePages := true
	for hasMorePages {
		result, err := tgw.Client.DescribeTransitGatewayVpcAttachments(ctx, params)
		if err != nil {
			logging.Debugf("[Transit Gateway] Failed to list transit gateway VPC attachments: %s", err)
			return nil, errors.WithStackTrace(err)
		}

		for _, tgwVpcAttachment := range result.TransitGatewayVpcAttachments {
			if configObj.TransitGatewaysVpcAttachment.ShouldInclude(config.ResourceValue{
				Time: tgwVpcAttachment.CreationTime,
			}) && tgwVpcAttachment.State != "deleted" && tgwVpcAttachment.State != "deleting" {
				identifiers = append(identifiers, tgwVpcAttachment.TransitGatewayAttachmentId)
			}
		}

		params.NextToken = result.NextToken
		hasMorePages = params.NextToken != nil
	}

	return identifiers, nil
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

		_, err := tgw.Client.DeleteTransitGatewayVpcAttachment(tgw.Context, param)

		// Record status of this resource
		e := report.Entry{
			Identifier:   aws.ToString(id),
			ResourceType: tgw.ResourceName(),
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

	if waiterr := waitForTransitGatewayAttachmentToBeDeleted(*tgw); waiterr != nil {
		return errors.WithStackTrace(waiterr)
	}
	logging.Debugf("[OK] %d Transit Gateway Vpc Attachment(s) deleted in %s", len(deletedIds), tgw.Region)
	return nil
}

func waitForTransitGatewayAttachmentToBeDeleted(tgw TransitGatewaysVpcAttachment) error {
	for i := 0; i < 30; i++ {
		gateways, err := tgw.Client.DescribeTransitGatewayVpcAttachments(
			tgw.Context, &ec2.DescribeTransitGatewayVpcAttachmentsInput{
				TransitGatewayAttachmentIds: tgw.Ids,
				Filters: []types.Filter{
					{
						Name:   aws.String("state"),
						Values: []string{"deleting"},
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
		logging.Info("Waiting for transit gateways attachments to be deleted...")
		time.Sleep(10 * time.Second)
	}

	return goerror.New("timed out waiting for transit gateway attachments to be successfully deleted")
}
