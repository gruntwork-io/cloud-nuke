package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

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
