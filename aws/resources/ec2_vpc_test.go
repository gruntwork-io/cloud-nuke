package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockEC2VpcClient struct {
	// Core VPC operations
	DescribeVpcsOutput ec2.DescribeVpcsOutput
	DeleteVpcOutput    ec2.DeleteVpcOutput
	DeleteVpcError     error

	// Safety net dependencies (ordered to match cleanupVPCDependencies execution order)
	DescribeVpcPeeringConnectionsOutput ec2.DescribeVpcPeeringConnectionsOutput
	DeleteVpcPeeringConnectionOutput    ec2.DeleteVpcPeeringConnectionOutput
	DescribeVpnGatewaysOutput           ec2.DescribeVpnGatewaysOutput
	DetachVpnGatewayOutput              ec2.DetachVpnGatewayOutput
	DeleteVpnGatewayOutput              ec2.DeleteVpnGatewayOutput
	DescribeRouteTablesOutput           ec2.DescribeRouteTablesOutput
	DisassociateRouteTableOutput        ec2.DisassociateRouteTableOutput
	DeleteRouteTableOutput              ec2.DeleteRouteTableOutput
	DescribeNetworkInterfacesOutput     ec2.DescribeNetworkInterfacesOutput
	DetachNetworkInterfaceOutput        ec2.DetachNetworkInterfaceOutput
	DeleteNetworkInterfaceOutput        ec2.DeleteNetworkInterfaceOutput
	DescribeSecurityGroupsOutput        ec2.DescribeSecurityGroupsOutput
	RevokeSecurityGroupIngressOutput    ec2.RevokeSecurityGroupIngressOutput
	RevokeSecurityGroupEgressOutput     ec2.RevokeSecurityGroupEgressOutput
	DeleteSecurityGroupOutput           ec2.DeleteSecurityGroupOutput
	DescribeSubnetsOutput               ec2.DescribeSubnetsOutput
	DeleteSubnetOutput                  ec2.DeleteSubnetOutput
	DescribeInternetGatewaysOutput      ec2.DescribeInternetGatewaysOutput
	DetachInternetGatewayOutput         ec2.DetachInternetGatewayOutput
	DeleteInternetGatewayOutput         ec2.DeleteInternetGatewayOutput

	// Track calls for assertions (matches execution order)
	DeletedPeeringIDs       []string
	DeletedVGWIDs           []string
	DeletedRouteTableIDs    []string
	DeletedENIIDs           []string
	DeletedSecurityGroupIDs []string
	DeletedSubnetIDs        []string
	DeletedIGWIDs           []string
}

func (m *mockEC2VpcClient) DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	return &m.DescribeVpcsOutput, nil
}

func (m *mockEC2VpcClient) DeleteVpc(ctx context.Context, params *ec2.DeleteVpcInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcOutput, error) {
	return &m.DeleteVpcOutput, m.DeleteVpcError
}

func (m *mockEC2VpcClient) CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error) {
	return &ec2.CreateTagsOutput{}, nil
}

func (m *mockEC2VpcClient) DescribeRouteTables(ctx context.Context, params *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error) {
	return &m.DescribeRouteTablesOutput, nil
}

func (m *mockEC2VpcClient) DisassociateRouteTable(ctx context.Context, params *ec2.DisassociateRouteTableInput, optFns ...func(*ec2.Options)) (*ec2.DisassociateRouteTableOutput, error) {
	return &m.DisassociateRouteTableOutput, nil
}

func (m *mockEC2VpcClient) DeleteRouteTable(ctx context.Context, params *ec2.DeleteRouteTableInput, optFns ...func(*ec2.Options)) (*ec2.DeleteRouteTableOutput, error) {
	m.DeletedRouteTableIDs = append(m.DeletedRouteTableIDs, aws.ToString(params.RouteTableId))
	return &m.DeleteRouteTableOutput, nil
}

func (m *mockEC2VpcClient) DescribeSecurityGroups(ctx context.Context, input *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	return &m.DescribeSecurityGroupsOutput, nil
}

func (m *mockEC2VpcClient) RevokeSecurityGroupIngress(ctx context.Context, input *ec2.RevokeSecurityGroupIngressInput, optFns ...func(*ec2.Options)) (*ec2.RevokeSecurityGroupIngressOutput, error) {
	return &m.RevokeSecurityGroupIngressOutput, nil
}

func (m *mockEC2VpcClient) RevokeSecurityGroupEgress(ctx context.Context, input *ec2.RevokeSecurityGroupEgressInput, optFns ...func(*ec2.Options)) (*ec2.RevokeSecurityGroupEgressOutput, error) {
	return &m.RevokeSecurityGroupEgressOutput, nil
}

func (m *mockEC2VpcClient) DeleteSecurityGroup(ctx context.Context, input *ec2.DeleteSecurityGroupInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSecurityGroupOutput, error) {
	m.DeletedSecurityGroupIDs = append(m.DeletedSecurityGroupIDs, aws.ToString(input.GroupId))
	return &m.DeleteSecurityGroupOutput, nil
}

func (m *mockEC2VpcClient) DescribeNetworkInterfaces(ctx context.Context, params *ec2.DescribeNetworkInterfacesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
	return &m.DescribeNetworkInterfacesOutput, nil
}

func (m *mockEC2VpcClient) DetachNetworkInterface(ctx context.Context, params *ec2.DetachNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.DetachNetworkInterfaceOutput, error) {
	return &m.DetachNetworkInterfaceOutput, nil
}

func (m *mockEC2VpcClient) DeleteNetworkInterface(ctx context.Context, params *ec2.DeleteNetworkInterfaceInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNetworkInterfaceOutput, error) {
	m.DeletedENIIDs = append(m.DeletedENIIDs, aws.ToString(params.NetworkInterfaceId))
	return &m.DeleteNetworkInterfaceOutput, nil
}

func (m *mockEC2VpcClient) DescribeInternetGateways(ctx context.Context, params *ec2.DescribeInternetGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInternetGatewaysOutput, error) {
	return &m.DescribeInternetGatewaysOutput, nil
}

func (m *mockEC2VpcClient) DetachInternetGateway(ctx context.Context, params *ec2.DetachInternetGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DetachInternetGatewayOutput, error) {
	return &m.DetachInternetGatewayOutput, nil
}

func (m *mockEC2VpcClient) DeleteInternetGateway(ctx context.Context, params *ec2.DeleteInternetGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteInternetGatewayOutput, error) {
	m.DeletedIGWIDs = append(m.DeletedIGWIDs, aws.ToString(params.InternetGatewayId))
	return &m.DeleteInternetGatewayOutput, nil
}

func (m *mockEC2VpcClient) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	return &m.DescribeSubnetsOutput, nil
}

func (m *mockEC2VpcClient) DeleteSubnet(ctx context.Context, params *ec2.DeleteSubnetInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSubnetOutput, error) {
	m.DeletedSubnetIDs = append(m.DeletedSubnetIDs, aws.ToString(params.SubnetId))
	return &m.DeleteSubnetOutput, nil
}

func (m *mockEC2VpcClient) DescribeVpcPeeringConnections(ctx context.Context, params *ec2.DescribeVpcPeeringConnectionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcPeeringConnectionsOutput, error) {
	return &m.DescribeVpcPeeringConnectionsOutput, nil
}

func (m *mockEC2VpcClient) DeleteVpcPeeringConnection(ctx context.Context, params *ec2.DeleteVpcPeeringConnectionInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcPeeringConnectionOutput, error) {
	m.DeletedPeeringIDs = append(m.DeletedPeeringIDs, aws.ToString(params.VpcPeeringConnectionId))
	return &m.DeleteVpcPeeringConnectionOutput, nil
}

func (m *mockEC2VpcClient) DescribeVpnGateways(ctx context.Context, params *ec2.DescribeVpnGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpnGatewaysOutput, error) {
	return &m.DescribeVpnGatewaysOutput, nil
}

func (m *mockEC2VpcClient) DetachVpnGateway(ctx context.Context, params *ec2.DetachVpnGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DetachVpnGatewayOutput, error) {
	return &m.DetachVpnGatewayOutput, nil
}

func (m *mockEC2VpcClient) DeleteVpnGateway(ctx context.Context, params *ec2.DeleteVpnGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpnGatewayOutput, error) {
	m.DeletedVGWIDs = append(m.DeletedVGWIDs, aws.ToString(params.VpnGatewayId))
	return &m.DeleteVpnGatewayOutput, nil
}

func TestListVPCs(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testId1 := "vpc-001"
	testId2 := "vpc-002"
	testName1 := "test-vpc-1"
	testName2 := "test-vpc-2"

	mock := &mockEC2VpcClient{
		DescribeVpcsOutput: ec2.DescribeVpcsOutput{
			Vpcs: []types.Vpc{
				{
					VpcId: aws.String(testId1),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName1)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					VpcId: aws.String(testId2),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName2)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
			},
		},
	}

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	tests := map[string]struct {
		cfg      config.ResourceType
		expected []string
	}{
		"emptyFilter": {
			cfg:      config.ResourceType{},
			expected: []string{testId1, testId2},
		},
		"nameExclusionFilter": {
			cfg: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testId2},
		},
		"timeAfterExclusionFilter": {
			cfg: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				},
			},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := listVPCs(ctx, mock, resource.Scope{Region: "us-east-1"}, tc.cfg, false)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestDeleteVPC(t *testing.T) {
	t.Parallel()

	mock := &mockEC2VpcClient{}
	err := deleteVPC(context.Background(), mock, aws.String("vpc-test"))
	require.NoError(t, err)
}

func TestCleanupVPCDependencies(t *testing.T) {
	t.Parallel()

	// Mock data ordered to match cleanupVPCDependencies execution order
	mock := &mockEC2VpcClient{
		DescribeVpcPeeringConnectionsOutput: ec2.DescribeVpcPeeringConnectionsOutput{
			VpcPeeringConnections: []types.VpcPeeringConnection{
				{VpcPeeringConnectionId: aws.String("pcx-1")},
			},
		},
		DescribeVpnGatewaysOutput: ec2.DescribeVpnGatewaysOutput{
			VpnGateways: []types.VpnGateway{
				{
					VpnGatewayId: aws.String("vgw-1"),
					VpcAttachments: []types.VpcAttachment{
						{VpcId: aws.String("vpc-test"), State: types.AttachmentStatusDetached},
					},
				},
			},
		},
		DescribeRouteTablesOutput: ec2.DescribeRouteTablesOutput{
			RouteTables: []types.RouteTable{
				{
					RouteTableId: aws.String("rtb-1"),
					Associations: []types.RouteTableAssociation{
						{RouteTableAssociationId: aws.String("rtbassoc-1"), Main: aws.Bool(false)},
					},
				},
				// Main route table — should be skipped
				{
					RouteTableId: aws.String("rtb-main"),
					Associations: []types.RouteTableAssociation{
						{RouteTableAssociationId: aws.String("rtbassoc-main"), Main: aws.Bool(true)},
					},
				},
				// Orphaned route table (no associations) — should be deleted
				{
					RouteTableId: aws.String("rtb-orphan"),
				},
			},
		},
		DescribeNetworkInterfacesOutput: ec2.DescribeNetworkInterfacesOutput{
			NetworkInterfaces: []types.NetworkInterface{
				{
					NetworkInterfaceId: aws.String("eni-1"),
					Attachment:         &types.NetworkInterfaceAttachment{AttachmentId: aws.String("attach-1")},
					Status:             types.NetworkInterfaceStatusAvailable,
				},
			},
		},
		DescribeSecurityGroupsOutput: ec2.DescribeSecurityGroupsOutput{
			SecurityGroups: []types.SecurityGroup{
				{GroupId: aws.String("sg-default"), GroupName: aws.String("default")},
				{GroupId: aws.String("sg-custom"), GroupName: aws.String("my-sg")},
			},
		},
		DescribeSubnetsOutput: ec2.DescribeSubnetsOutput{
			Subnets: []types.Subnet{
				{SubnetId: aws.String("subnet-1")},
				{SubnetId: aws.String("subnet-2")},
			},
		},
		DescribeInternetGatewaysOutput: ec2.DescribeInternetGatewaysOutput{
			InternetGateways: []types.InternetGateway{
				{InternetGatewayId: aws.String("igw-1")},
			},
		},
	}

	err := cleanupVPCDependencies(context.Background(), mock, aws.String("vpc-test"))
	require.NoError(t, err)

	// Verify cleanup actions (ordered to match execution order)
	require.Equal(t, []string{"pcx-1", "pcx-1"}, mock.DeletedPeeringIDs)         // appears twice: requester + accepter queries
	require.Equal(t, []string{"vgw-1"}, mock.DeletedVGWIDs)                      // VPN gateway detached + deleted
	require.Equal(t, []string{"rtb-1", "rtb-orphan"}, mock.DeletedRouteTableIDs) // main RT skipped, orphan included
	require.Equal(t, []string{"eni-1"}, mock.DeletedENIIDs)
	require.Equal(t, []string{"sg-custom"}, mock.DeletedSecurityGroupIDs) // default SG skipped
	require.Equal(t, []string{"subnet-1", "subnet-2"}, mock.DeletedSubnetIDs)
	require.Equal(t, []string{"igw-1"}, mock.DeletedIGWIDs)
}
