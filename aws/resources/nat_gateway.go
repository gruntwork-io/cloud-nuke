package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// NatGatewaysAPI defines the interface for NAT Gateway operations.
type NatGatewaysAPI interface {
	DeleteNatGateway(ctx context.Context, params *ec2.DeleteNatGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNatGatewayOutput, error)
	DescribeNatGateways(ctx context.Context, params *ec2.DescribeNatGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNatGatewaysOutput, error)
}

// NewNatGateways creates a new NatGateways resource using the generic resource pattern.
func NewNatGateways() AwsResource {
	return NewAwsResource(&resource.Resource[NatGatewaysAPI]{
		ResourceTypeName: "nat-gateway",
		BatchSize:        10,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[NatGatewaysAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.NatGateway
		},
		Lister: listNatGateways,
		Nuker:  resource.SimpleBatchDeleter(deleteNatGateway),
	})
}

// listNatGateways retrieves all NAT Gateways that match the config filters.
func listNatGateways(ctx context.Context, client NatGatewaysAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var allNatGateways []*string

	paginator := ec2.NewDescribeNatGatewaysPaginator(client, &ec2.DescribeNatGatewaysInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, gateway := range page.NatGateways {
			if shouldIncludeNatGateway(gateway, cfg) {
				allNatGateways = append(allNatGateways, gateway.NatGatewayId)
			}
		}
	}

	return allNatGateways, nil
}

func shouldIncludeNatGateway(ngw types.NatGateway, cfg config.ResourceType) bool {
	if ngw.State == types.NatGatewayStateDeleted || ngw.State == types.NatGatewayStateDeleting {
		return false
	}

	return cfg.ShouldInclude(config.ResourceValue{
		Time: ngw.CreateTime,
		Name: getNatGatewayName(ngw),
		Tags: util.ConvertTypesTagsToMap(ngw.Tags),
	})
}

func getNatGatewayName(ngw types.NatGateway) *string {
	for _, tag := range ngw.Tags {
		if aws.ToString(tag.Key) == "Name" {
			return tag.Value
		}
	}
	return nil
}

// deleteNatGateway deletes a single NAT Gateway.
func deleteNatGateway(ctx context.Context, client NatGatewaysAPI, id *string) error {
	_, err := client.DeleteNatGateway(ctx, &ec2.DeleteNatGatewayInput{
		NatGatewayId: id,
	})
	return err
}
