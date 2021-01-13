package aws

import (
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Returns a formatted string of TranstGatewayRouteTable IDs
func getAllTransitGatewayRouteTables(session *session.Session, region string, excludeAfter time.Time) ([]*string, error) {
	svc := ec2.New(session)
	result, err := svc.DescribeTransitGatewayRouteTables(&ec2.DescribeTransitGatewayRouteTablesInput{})
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
