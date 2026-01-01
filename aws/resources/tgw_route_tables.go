package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
)

// TransitGatewaysRouteTablesAPI defines the interface for Transit Gateway Route Tables operations.
type TransitGatewaysRouteTablesAPI interface {
	DescribeTransitGatewayRouteTables(ctx context.Context, params *ec2.DescribeTransitGatewayRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayRouteTablesOutput, error)
	DeleteTransitGatewayRouteTable(ctx context.Context, params *ec2.DeleteTransitGatewayRouteTableInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayRouteTableOutput, error)
}

// NewTransitGatewaysRouteTables creates a new TransitGatewaysRouteTables resource using the generic resource pattern.
func NewTransitGatewaysRouteTables() AwsResource {
	return NewAwsResource(&resource.Resource[TransitGatewaysRouteTablesAPI]{
		ResourceTypeName: "transit-gateway-route-table",
		BatchSize:        maxBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[TransitGatewaysRouteTablesAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.TransitGatewayRouteTable
		},
		Lister: listTransitGatewayRouteTables,
		Nuker:  resource.SimpleBatchDeleter(deleteTransitGatewayRouteTable),
	})
}

// listTransitGatewayRouteTables retrieves all Transit Gateway Route Tables that match the config filters.
func listTransitGatewayRouteTables(ctx context.Context, client TransitGatewaysRouteTablesAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	// Remove default route table, that will be deleted along with its TransitGateway
	param := &ec2.DescribeTransitGatewayRouteTablesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("default-association-route-table"),
				Values: []string{"false"},
			},
		},
	}

	result, err := client.DescribeTransitGatewayRouteTables(ctx, param)
	if err != nil {
		return nil, err
	}

	var ids []*string
	for _, transitGatewayRouteTable := range result.TransitGatewayRouteTables {
		if cfg.ShouldInclude(config.ResourceValue{Time: transitGatewayRouteTable.CreationTime}) &&
			transitGatewayRouteTable.State != types.TransitGatewayRouteTableStateDeleted &&
			transitGatewayRouteTable.State != types.TransitGatewayRouteTableStateDeleting {
			ids = append(ids, transitGatewayRouteTable.TransitGatewayRouteTableId)
		}
	}

	return ids, nil
}

// deleteTransitGatewayRouteTable deletes a single Transit Gateway Route Table.
func deleteTransitGatewayRouteTable(ctx context.Context, client TransitGatewaysRouteTablesAPI, id *string) error {
	_, err := client.DeleteTransitGatewayRouteTable(ctx, &ec2.DeleteTransitGatewayRouteTableInput{
		TransitGatewayRouteTableId: id,
	})
	return err
}
