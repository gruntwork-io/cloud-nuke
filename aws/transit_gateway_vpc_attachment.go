package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Returns a formated string of TransitGatewayVpcAttachment IDs
func getAllTransitGatewayVpcAttachments(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := ec2.New(session)
	result, err := svc.DescribeTransitGatewayVpcAttachments(&ec2.DescribeTransitGatewayVpcAttachmentsInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var ids []*string
	for _, tgwVpcAttachment := range result.TransitGatewayVpcAttachments {
		if excludeAfter.After(*tgwVpcAttachment.CreationTime) && awsgo.StringValue(tgwVpcAttachment.State) != "deleted" && awsgo.StringValue(tgwVpcAttachment.State) != "deleting" {
			ids = append(ids, tgwVpcAttachment.TransitGatewayAttachmentId)
		}
	}

	return ids, nil
}

// Delete all TransitGatewayVpcAttachments
func nukeAllTransitGatewayVpcAttachments(session *session.Session, ids []*string) error {
	svc := ec2.New(session)

	if len(ids) == 0 {
		logging.Logger.Infof("No Transit Gateway Vpc Attachments to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all Transit Gateway Vpc Attachments in region %s", *session.Config.Region)
	var deletedIds []*string

	for _, id := range ids {
		param := &ec2.DeleteTransitGatewayVpcAttachmentInput{
			TransitGatewayAttachmentId: id,
		}

		_, err := svc.DeleteTransitGatewayVpcAttachment(param)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			deletedIds = append(deletedIds, id)
			logging.Logger.Infof("Deleted Transit Gateway Vpc Attachment: %s", *id)
		}
	}

	//TransitGatewayAttachment takes some time to be available and there isn't Waiters available yet
	//To avoid test errors, I'm introducing a sleep call
	time.Sleep(180 * time.Second)
	logging.Logger.Infof(("[OK] %d Transit Gateway Vpc Attachment(s) deleted in %s"), len(deletedIds), *session.Config.Region)
	return nil
}
