package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

func (tgpa *TransitGatewayPeeringAttachment) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	var ids []*string
	paginator := ec2.NewDescribeTransitGatewayPeeringAttachmentsPaginator(tgpa.Client, &ec2.DescribeTransitGatewayPeeringAttachmentsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(c)
		if err != nil {
			return nil, errors.WithStackTrace(err)
		}

		for _, attachment := range page.TransitGatewayPeeringAttachments {
			if configObj.TransitGatewayPeeringAttachment.ShouldInclude(config.ResourceValue{
				Time: attachment.CreationTime,
			}) {
				ids = append(ids, attachment.TransitGatewayAttachmentId)
			}
		}
	}

	return ids, nil
}

func (tgpa *TransitGatewayPeeringAttachment) nukeAll(ids []*string) error {
	for _, id := range ids {
		_, err := tgpa.Client.DeleteTransitGatewayPeeringAttachment(tgpa.Context, &ec2.DeleteTransitGatewayPeeringAttachmentInput{
			TransitGatewayAttachmentId: id,
		})
		// Record status of this resource
		report.Record(report.Entry{
			Identifier:   aws.ToString(id),
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
