package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EC2VPCAPI interface {
	DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	DeleteVpc(ctx context.Context, params *ec2.DeleteVpcInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcOutput, error)
	DescribeVpcEndpointServiceConfigurations(ctx context.Context, params *ec2.DescribeVpcEndpointServiceConfigurationsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointServiceConfigurationsOutput, error)
	DeleteVpcEndpointServiceConfigurations(ctx context.Context, params *ec2.DeleteVpcEndpointServiceConfigurationsInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcEndpointServiceConfigurationsOutput, error)
	DeleteVpcPeeringConnection(ctx context.Context, params *ec2.DeleteVpcPeeringConnectionInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcPeeringConnectionOutput, error)
	DescribeVpcEndpoints(ctx context.Context, params *ec2.DescribeVpcEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointsOutput, error)
	DescribeRouteTables(ctx context.Context, params *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error)
	DisassociateRouteTable(ctx context.Context, params *ec2.DisassociateRouteTableInput, optFns ...func(*ec2.Options)) (*ec2.DisassociateRouteTableOutput, error)
	DeleteRouteTable(ctx context.Context, params *ec2.DeleteRouteTableInput, optFns ...func(*ec2.Options)) (*ec2.DeleteRouteTableOutput, error)
	DescribeSecurityGroupRules(ctx context.Context, params *ec2.DescribeSecurityGroupRulesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupRulesOutput, error)
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	DescribeVpcPeeringConnections(ctx context.Context, params *ec2.DescribeVpcPeeringConnectionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcPeeringConnectionsOutput, error)
	AcceptAddressTransfer(ctx context.Context, params *ec2.AcceptAddressTransferInput, optFns ...func(*ec2.Options)) (*ec2.AcceptAddressTransferOutput, error)
	NetworkInterfaceAPI
	EC2DhcpOptionAPI
	NetworkACLAPI
	EC2EndpointsAPI
	NetworkACLAPI
	InternetGatewayAPI
	EgressOnlyIGAPI
	EC2SubnetAPI
	NatGatewaysAPI
	SecurityGroupAPI
}
type ELBClientAPI interface {
	DescribeLoadBalancers(ctx context.Context, input *elbv2.DescribeLoadBalancersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error)
	DeleteLoadBalancer(ctx context.Context, input *elbv2.DeleteLoadBalancerInput, optFns ...func(*elbv2.Options)) (*elbv2.DeleteLoadBalancerOutput, error)
	DescribeTargetGroups(ctx context.Context, input *elbv2.DescribeTargetGroupsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error)
	DeleteTargetGroup(ctx context.Context, input *elbv2.DeleteTargetGroupInput, optFns ...func(*elbv2.Options)) (*elbv2.DeleteTargetGroupOutput, error)
}
type EC2VPCs struct {
	BaseAwsResource
	Client    EC2VPCAPI
	ELBClient ELBClientAPI
	Region    string
	VPCIds    []string
}

func (v *EC2VPCs) Init(cfg aws.Config) {
	v.Client = ec2.NewFromConfig(cfg)
	v.ELBClient = elbv2.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (v *EC2VPCs) ResourceName() string {
	return "vpc"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (v *EC2VPCs) ResourceIdentifiers() []string {
	return v.VPCIds
}

func (v *EC2VPCs) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// func (v *EC2VPCs) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
// 	return configObj.VPC
// }

func (v *EC2VPCs) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := v.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	v.VPCIds = aws.ToStringSlice(identifiers)
	return v.VPCIds, nil
}

// Nuke - nuke 'em all!!!
func (v *EC2VPCs) Nuke(ctx context.Context, identifiers []string) error {
	if err := v.nukeAll(identifiers); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
