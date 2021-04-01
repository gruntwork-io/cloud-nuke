package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

func sleepWithMessage(duration time.Duration, whySleepMessage string) {
	logging.Logger.Infof("Sleeping %v: %s", duration, whySleepMessage)
	time.Sleep(duration)
}

// Returns a formatted string of TransitGateway IDs
func getAllTransitGatewayInstances(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := ec2.New(session)
	result, err := svc.DescribeTransitGateways(&ec2.DescribeTransitGatewaysInput{})
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var ids []*string
	for _, transitGateway := range result.TransitGateways {
		if excludeAfter.After(*transitGateway.CreationTime) && awsgo.StringValue(transitGateway.State) != "deleted" && awsgo.StringValue(transitGateway.State) != "deleting" {
			ids = append(ids, transitGateway.TransitGatewayId)
		}
	}

	return ids, nil
}

// Delete all TransitGateways
func nukeAllTransitGatewayInstances(session *session.Session, ids []*string) error {
	svc := ec2.New(session)

	if len(ids) == 0 {
		logging.Logger.Infof("No Transit Gateways to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all Transit Gateways in region %s", *session.Config.Region)
	var deletedIds []*string

	for _, id := range ids {
		params := &ec2.DeleteTransitGatewayInput{
			TransitGatewayId: id,
		}

		_, err := svc.DeleteTransitGateway(params)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			deletedIds = append(deletedIds, id)
			logging.Logger.Infof("Deleted Transit Gateway: %s", *id)
		}
	}

	logging.Logger.Infof("[OK] %d Transit Gateway(s) deleted in %s", len(deletedIds), *session.Config.Region)
	return nil
}

// Returns a formatted string of TranstGatewayRouteTable IDs
func getAllTransitGatewayRouteTables(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := ec2.New(session)

	// Remove defalt route table, that will be deleted along with its TransitGateway
	param := &ec2.DescribeTransitGatewayRouteTablesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("default-association-route-table"),
				Values: []*string{
					aws.String("false"),
				}},
		},
	}

	result, err := svc.DescribeTransitGatewayRouteTables(param)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	var ids []*string
	for _, transitGatewayRouteTable := range result.TransitGatewayRouteTables {
		if excludeAfter.After(*transitGatewayRouteTable.CreationTime) && awsgo.StringValue(transitGatewayRouteTable.State) != "deleted" && awsgo.StringValue(transitGatewayRouteTable.State) != "deleting" {
			ids = append(ids, transitGatewayRouteTable.TransitGatewayRouteTableId)
		}
	}

	return ids, nil
}

// Delete all TransitGatewayRouteTables
func nukeAllTransitGatewayRouteTables(session *session.Session, ids []*string) error {
	svc := ec2.New(session)

	if len(ids) == 0 {
		logging.Logger.Infof("No Transit Gateway Route Tables to nuke in region %s", *session.Config.Region)
		return nil
	}

	logging.Logger.Infof("Deleting all Transit Gateway Route Tables in region %s", *session.Config.Region)
	var deletedIds []*string

	for _, id := range ids {
		param := &ec2.DeleteTransitGatewayRouteTableInput{
			TransitGatewayRouteTableId: id,
		}

		_, err := svc.DeleteTransitGatewayRouteTable(param)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		} else {
			deletedIds = append(deletedIds, id)
			logging.Logger.Infof("Deleted Transit Gateway Route Table: %s", *id)
		}
	}

	logging.Logger.Infof("[OK] %d Transit Gateway Route Table(s) deleted in %s", len(deletedIds), *session.Config.Region)
	return nil
}

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

	sleepMessage := "TransitGateway Vpc Attachments takes some time to create, and since there is no waiter available, we sleep instead."
	sleepFor := 180 * time.Second
	sleepWithMessage(sleepFor, sleepMessage)

	logging.Logger.Infof(("[OK] %d Transit Gateway Vpc Attachment(s) deleted in %s"), len(deletedIds), *session.Config.Region)
	return nil
}

func tgIsAvailableInRegion(session *session.Session, region string) (bool, error) {
	svc := ec2.New(session)
	_, err := svc.DescribeTransitGateways(&ec2.DescribeTransitGatewaysInput{})
	if err != nil {
		if awsErr, isAwsErr := err.(awserr.Error); isAwsErr && awsErr.Code() == "InvalidAction" {
			return false, nil
		} else {
			return nil, errors.WithStackTrace(err)
		}
	}
	return true, nil
}
