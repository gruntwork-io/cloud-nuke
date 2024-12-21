package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type TransitGatewaysRouteTablesAPI interface {
	DeleteTransitGatewayRouteTable(ctx context.Context, params *ec2.DeleteTransitGatewayRouteTableInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayRouteTableOutput, error)
	DescribeTransitGatewayRouteTables(ctx context.Context, params *ec2.DescribeTransitGatewayRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayRouteTablesOutput, error)
}

// TransitGatewaysRouteTables - represents all transit gateways route tables
type TransitGatewaysRouteTables struct {
	BaseAwsResource
	Client TransitGatewaysRouteTablesAPI
	Region string
	Ids    []string
}

func (tgw *TransitGatewaysRouteTables) InitV2(cfg aws.Config) {
	tgw.Client = ec2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (tgw *TransitGatewaysRouteTables) ResourceName() string {
	return "transit-gateway-route-table"
}

// MaxBatchSize - Tentative batch size to ensure AWS doesn't throttle
func (tgw *TransitGatewaysRouteTables) MaxBatchSize() int {
	return maxBatchSize
}

// ResourceIdentifiers - The arns of the transit gateways route tables
func (tgw *TransitGatewaysRouteTables) ResourceIdentifiers() []string {
	return tgw.Ids
}

func (tgw *TransitGatewaysRouteTables) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := tgw.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	tgw.Ids = aws.ToStringSlice(identifiers)
	return tgw.Ids, nil
}

// Nuke - nuke 'em all!!!
func (tgw *TransitGatewaysRouteTables) Nuke(identifiers []string) error {
	if err := tgw.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
