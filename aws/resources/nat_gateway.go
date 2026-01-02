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
	DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
}

// NewNatGateways creates a new NatGateways resource using the generic resource pattern.
func NewNatGateways() AwsResource {
	return NewEC2AwsResource[NatGatewaysAPI](
		"nat-gateway",
		WrapAwsInitClient(func(r *resource.Resource[NatGatewaysAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		func(c config.Config) config.EC2ResourceType { return c.NatGateway },
		listNatGateways,
		resource.SimpleBatchDeleter(deleteNatGateway),
		nil,
	)
}

// listNatGateways retrieves all NAT Gateways that match the config filters.
func listNatGateways(ctx context.Context, client NatGatewaysAPI, scope resource.Scope, cfg config.ResourceType, defaultOnly bool) ([]*string, error) {
	// When defaultOnly is true, get the list of default VPC IDs to filter by
	var defaultVpcIds map[string]bool
	if defaultOnly {
		defaultVpcIds = make(map[string]bool)
		vpcs, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
			Filters: []types.Filter{
				{Name: aws.String("is-default"), Values: []string{"true"}},
			},
		})
		if err != nil {
			return nil, err
		}
		for _, vpc := range vpcs.Vpcs {
			defaultVpcIds[aws.ToString(vpc.VpcId)] = true
		}
		if len(defaultVpcIds) == 0 {
			return nil, nil
		}
	}

	var allNatGateways []*string

	paginator := ec2.NewDescribeNatGatewaysPaginator(client, &ec2.DescribeNatGatewaysInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, gateway := range page.NatGateways {
			// When defaultOnly is true, skip NAT gateways not in default VPCs
			if defaultOnly && !defaultVpcIds[aws.ToString(gateway.VpcId)] {
				continue
			}

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
