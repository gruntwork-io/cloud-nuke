package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/report"
	"github.com/gruntwork-io/go-commons/errors"
)

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
