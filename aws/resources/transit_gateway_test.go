package resources

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedTransitGateway struct {
	ec2iface.EC2API
	DescribeTransitGatewaysOutput ec2.DescribeTransitGatewaysOutput
	DeleteTransitGatewayOutput    ec2.DeleteTransitGatewayOutput
}

type mockedTransitGatewayRouteTable struct {
	ec2iface.EC2API
	DescribeTransitGatewayRouteTablesOutput ec2.DescribeTransitGatewayRouteTablesOutput
	DeleteTransitGatewayRouteTableOutput    ec2.DeleteTransitGatewayRouteTableOutput
}

type mockedTransitGatewayVpcAttachment struct {
	ec2iface.EC2API
	DescribeTransitGatewayVpcAttachmentsOutput ec2.DescribeTransitGatewayVpcAttachmentsOutput
	DeleteTransitGatewayVpcAttachmentOutput    ec2.DeleteTransitGatewayVpcAttachmentOutput
}

type mockedTransitGatewayPeeringAttachment struct {
	ec2iface.EC2API
	DescribeTransitGatewayPeeringAttachmentsOutput ec2.DescribeTransitGatewayPeeringAttachmentsOutput
	DeleteTransitGatewayPeeringAttachmentOutput    ec2.DeleteTransitGatewayPeeringAttachmentOutput
}

func (m mockedTransitGatewayPeeringAttachment) DescribeTransitGatewayPeeringAttachmentsPagesWithContext(_ awsgo.Context, _ *ec2.DescribeTransitGatewayPeeringAttachmentsInput, fn func(*ec2.DescribeTransitGatewayPeeringAttachmentsOutput, bool) bool, _ ...request.Option) error {
	fn(&m.DescribeTransitGatewayPeeringAttachmentsOutput, true)
	return nil
}

func (m mockedTransitGatewayPeeringAttachment) DeleteTransitGatewayPeeringAttachmentWithContext(_ awsgo.Context, _ *ec2.DeleteTransitGatewayPeeringAttachmentInput, _ ...request.Option) (*ec2.DeleteTransitGatewayPeeringAttachmentOutput, error) {
	return &m.DeleteTransitGatewayPeeringAttachmentOutput, nil
}

func (m mockedTransitGateway) DescribeTransitGatewaysWithContext(_ awsgo.Context, _ *ec2.DescribeTransitGatewaysInput, _ ...request.Option) (*ec2.DescribeTransitGatewaysOutput, error) {
	return &m.DescribeTransitGatewaysOutput, nil
}

func (m mockedTransitGateway) DeleteTransitGatewayWithContext(_ awsgo.Context, _ *ec2.DeleteTransitGatewayInput, _ ...request.Option) (*ec2.DeleteTransitGatewayOutput, error) {
	return &m.DeleteTransitGatewayOutput, nil
}

func (m mockedTransitGatewayRouteTable) DescribeTransitGatewayRouteTablesWithContext(_ awsgo.Context, _ *ec2.DescribeTransitGatewayRouteTablesInput, _ ...request.Option) (*ec2.DescribeTransitGatewayRouteTablesOutput, error) {
	return &m.DescribeTransitGatewayRouteTablesOutput, nil
}

func (m mockedTransitGatewayRouteTable) DeleteTransitGatewayRouteTableWithContext(_ awsgo.Context, _ *ec2.DeleteTransitGatewayRouteTableInput, _ ...request.Option) (*ec2.DeleteTransitGatewayRouteTableOutput, error) {
	return &m.DeleteTransitGatewayRouteTableOutput, nil
}

func (m mockedTransitGatewayVpcAttachment) DescribeTransitGatewayVpcAttachmentsWithContext(_ awsgo.Context, _ *ec2.DescribeTransitGatewayVpcAttachmentsInput, _ ...request.Option) (*ec2.DescribeTransitGatewayVpcAttachmentsOutput, error) {
	return &m.DescribeTransitGatewayVpcAttachmentsOutput, nil
}
func (m mockedTransitGatewayVpcAttachment) DescribeTransitGatewayVpcAttachments(_ *ec2.DescribeTransitGatewayVpcAttachmentsInput) (*ec2.DescribeTransitGatewayVpcAttachmentsOutput, error) {
	return &m.DescribeTransitGatewayVpcAttachmentsOutput, nil
}

func (m mockedTransitGatewayVpcAttachment) DeleteTransitGatewayVpcAttachmentWithContext(_ awsgo.Context, _ *ec2.DeleteTransitGatewayVpcAttachmentInput, _ ...request.Option) (*ec2.DeleteTransitGatewayVpcAttachmentOutput, error) {
	return &m.DeleteTransitGatewayVpcAttachmentOutput, nil
}

func TestTransitGateways_GetAll(t *testing.T) {

	t.Parallel()

	now := time.Now()
	gatewayId1 := "gateway1"
	gatewayId2 := "gateway2"
	tgw := TransitGateways{
		Client: mockedTransitGateway{
			DescribeTransitGatewaysOutput: ec2.DescribeTransitGatewaysOutput{
				TransitGateways: []*ec2.TransitGateway{
					{
						TransitGatewayId: aws.String(gatewayId1),
						CreationTime:     aws.Time(now),
						State:            aws.String("available"),
					},
					{
						TransitGatewayId: aws.String(gatewayId2),
						CreationTime:     aws.Time(now.Add(1)),
						State:            aws.String("deleting"),
					},
				},
			},
		},
	}
	tgw.BaseAwsResource.Init(nil)

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{gatewayId1},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := tgw.getAll(context.Background(), config.Config{
				TransitGateway: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}

}

func TestTransitGateways_NukeAll(t *testing.T) {

	t.Parallel()

	tgw := TransitGateways{
		Client: mockedTransitGateway{
			DeleteTransitGatewayOutput: ec2.DeleteTransitGatewayOutput{},
		},
	}

	err := tgw.nukeAll([]*string{aws.String("test-gateway")})
	require.NoError(t, err)
}

func TestTransitGatewayRouteTables_GetAll(t *testing.T) {

	t.Parallel()

	now := time.Now()
	tableId1 := "table1"
	tableId2 := "table2"
	tgw := TransitGatewaysRouteTables{
		Client: mockedTransitGatewayRouteTable{
			DescribeTransitGatewayRouteTablesOutput: ec2.DescribeTransitGatewayRouteTablesOutput{
				TransitGatewayRouteTables: []*ec2.TransitGatewayRouteTable{
					{
						TransitGatewayRouteTableId: aws.String(tableId1),
						CreationTime:               aws.Time(now),
						State:                      aws.String("available"),
					},
					{
						TransitGatewayRouteTableId: aws.String(tableId2),
						CreationTime:               aws.Time(now.Add(1)),
						State:                      aws.String("deleting"),
					},
				},
			},
		},
	}
	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{tableId1},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := tgw.getAll(context.Background(), config.Config{
				TransitGatewayRouteTable: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestTransitGatewayRouteTables_NukeAll(t *testing.T) {

	t.Parallel()

	tgw := TransitGatewaysRouteTables{
		Client: mockedTransitGatewayRouteTable{
			DeleteTransitGatewayRouteTableOutput: ec2.DeleteTransitGatewayRouteTableOutput{},
		},
	}

	err := tgw.nukeAll([]*string{aws.String("test-route-table")})
	require.NoError(t, err)
}

func TestTransitGatewayVpcAttachments_GetAll(t *testing.T) {

	t.Parallel()

	now := time.Now()
	attachment1 := "attachement1"
	attachment2 := "attachement2"
	tgw := TransitGatewaysVpcAttachment{
		Client: mockedTransitGatewayVpcAttachment{
			DescribeTransitGatewayVpcAttachmentsOutput: ec2.DescribeTransitGatewayVpcAttachmentsOutput{
				TransitGatewayVpcAttachments: []*ec2.TransitGatewayVpcAttachment{
					{
						TransitGatewayAttachmentId: aws.String(attachment1),
						CreationTime:               aws.Time(now),
						State:                      aws.String("available"),
					},
					{
						TransitGatewayAttachmentId: aws.String(attachment2),
						CreationTime:               aws.Time(now.Add(1)),
						State:                      aws.String("deleting"),
					},
				},
			},
		},
	}
	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{attachment1},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := tgw.getAll(context.Background(), config.Config{
				TransitGatewaysVpcAttachment: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestTransitGatewayVpcAttachments_NukeAll(t *testing.T) {

	t.Parallel()

	tgw := TransitGatewaysVpcAttachment{
		Client: mockedTransitGatewayVpcAttachment{
			DeleteTransitGatewayVpcAttachmentOutput: ec2.DeleteTransitGatewayVpcAttachmentOutput{},
		},
	}

	err := tgw.nukeAll([]*string{aws.String("test-attachment")})
	require.NoError(t, err)
}

func TestTransitGatewayPeeringAttachment_getAll(t *testing.T) {

	t.Parallel()

	now := time.Now()
	attachment1 := "attachement1"
	attachment2 := "attachement2"
	tgpa := TransitGatewayPeeringAttachment{
		Client: mockedTransitGatewayPeeringAttachment{
			DescribeTransitGatewayPeeringAttachmentsOutput: ec2.DescribeTransitGatewayPeeringAttachmentsOutput{
				TransitGatewayPeeringAttachments: []*ec2.TransitGatewayPeeringAttachment{
					{
						TransitGatewayAttachmentId: aws.String(attachment1),
						CreationTime:               aws.Time(now),
					},
					{
						TransitGatewayAttachmentId: aws.String(attachment2),
						CreationTime:               aws.Time(now.Add(1)),
					},
				},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{attachment1, attachment2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{attachment1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := tgpa.getAll(context.Background(), config.Config{
				TransitGatewayPeeringAttachment: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestTransitGatewayPeeringAttachment_nukeAll(t *testing.T) {

	t.Parallel()

	tgw := TransitGatewayPeeringAttachment{
		Client: mockedTransitGatewayPeeringAttachment{
			DeleteTransitGatewayPeeringAttachmentOutput: ec2.DeleteTransitGatewayPeeringAttachmentOutput{},
		},
	}

	err := tgw.nukeAll([]*string{aws.String("test-attachment")})
	require.NoError(t, err)
}
