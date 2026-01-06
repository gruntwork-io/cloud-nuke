package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// EgressOnlyIGAPI defines the interface for Egress Only Internet Gateway operations.
type EgressOnlyIGAPI interface {
	DescribeEgressOnlyInternetGateways(ctx context.Context, params *ec2.DescribeEgressOnlyInternetGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeEgressOnlyInternetGatewaysOutput, error)
	DeleteEgressOnlyInternetGateway(ctx context.Context, params *ec2.DeleteEgressOnlyInternetGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteEgressOnlyInternetGatewayOutput, error)
}

// NewEgressOnlyInternetGateway creates a new Egress Only Internet Gateway resource using the generic resource pattern.
func NewEgressOnlyInternetGateway() AwsResource {
	return NewAwsResource(&resource.Resource[EgressOnlyIGAPI]{
		ResourceTypeName: "egress-only-internet-gateway",
		BatchSize:        DefaultBatchSize,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[EgressOnlyIGAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.EgressOnlyInternetGateway
		},
		Lister:             listEgressOnlyInternetGateways,
		Nuker:              resource.SimpleBatchDeleter(deleteEgressOnlyInternetGateway),
		PermissionVerifier: verifyEgressOnlyInternetGatewayPermission,
	})
}

// listEgressOnlyInternetGateways retrieves all Egress Only Internet Gateways that match the config filters.
func listEgressOnlyInternetGateways(ctx context.Context, client EgressOnlyIGAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var gatewayIds []*string

	paginator := ec2.NewDescribeEgressOnlyInternetGatewaysPaginator(client, &ec2.DescribeEgressOnlyInternetGatewaysInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, gateway := range page.EgressOnlyInternetGateways {
			if cfg.ShouldInclude(config.ResourceValue{
				Name: util.GetEC2ResourceNameTagValue(gateway.Tags),
				Tags: util.ConvertTypesTagsToMap(gateway.Tags),
			}) {
				gatewayIds = append(gatewayIds, gateway.EgressOnlyInternetGatewayId)
			}
		}
	}

	return gatewayIds, nil
}

// verifyEgressOnlyInternetGatewayPermission performs a dry-run delete to check permissions.
func verifyEgressOnlyInternetGatewayPermission(ctx context.Context, client EgressOnlyIGAPI, id *string) error {
	_, err := client.DeleteEgressOnlyInternetGateway(ctx, &ec2.DeleteEgressOnlyInternetGatewayInput{
		EgressOnlyInternetGatewayId: id,
		DryRun:                      aws.Bool(true),
	})
	return err
}

// deleteEgressOnlyInternetGateway deletes a single Egress Only Internet Gateway.
func deleteEgressOnlyInternetGateway(ctx context.Context, client EgressOnlyIGAPI, id *string) error {
	_, err := client.DeleteEgressOnlyInternetGateway(ctx, &ec2.DeleteEgressOnlyInternetGatewayInput{
		EgressOnlyInternetGatewayId: id,
	})
	return err
}
