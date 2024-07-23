package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/go-commons/errors"
)

// Returns a formatted string of TransitGatewayRouteTable IDs
func (tgw *TransitGatewaysRouteTables) getAll(c context.Context, configObj config.Config) ([]*string, error) {
	// Remove default route table, that will be deleted along with its TransitGateway
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
