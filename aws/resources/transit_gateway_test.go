package resources

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedTransitGateway struct {
	TransitGatewayAPI
	DescribeTransitGatewaysOutput               ec2.DescribeTransitGatewaysOutput
	DeleteTransitGatewayOutput                  ec2.DeleteTransitGatewayOutput
	DescribeTransitGatewayAttachmentsOutput     ec2.DescribeTransitGatewayAttachmentsOutput
	DeleteTransitGatewayPeeringAttachmentOutput ec2.DeleteTransitGatewayPeeringAttachmentOutput
	DeleteTransitGatewayVpcAttachmentOutput     ec2.DeleteTransitGatewayVpcAttachmentOutput
	DeleteVpnConnectionOutput                   ec2.DeleteVpnConnectionOutput
	DeleteTransitGatewayConnectOutput           ec2.DeleteTransitGatewayConnectOutput
}

func (m mockedTransitGateway) DescribeTransitGateways(ctx context.Context, params *ec2.DescribeTransitGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewaysOutput, error) {
	return &m.DescribeTransitGatewaysOutput, nil
}

func (m mockedTransitGateway) DeleteTransitGateway(ctx context.Context, params *ec2.DeleteTransitGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayOutput, error) {
	return &m.DeleteTransitGatewayOutput, nil
}

func (m mockedTransitGateway) DescribeTransitGatewayAttachments(ctx context.Context, params *ec2.DescribeTransitGatewayAttachmentsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayAttachmentsOutput, error) {
	return &m.DescribeTransitGatewayAttachmentsOutput, nil
}

func (m mockedTransitGateway) DeleteTransitGatewayPeeringAttachment(ctx context.Context, params *ec2.DeleteTransitGatewayPeeringAttachmentInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayPeeringAttachmentOutput, error) {
	return &m.DeleteTransitGatewayPeeringAttachmentOutput, nil
}

func (m mockedTransitGateway) DeleteTransitGatewayVpcAttachment(ctx context.Context, params *ec2.DeleteTransitGatewayVpcAttachmentInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayVpcAttachmentOutput, error) {
	return &m.DeleteTransitGatewayVpcAttachmentOutput, nil
}

func (m mockedTransitGateway) DeleteTransitGatewayConnect(ctx context.Context, params *ec2.DeleteTransitGatewayConnectInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayConnectOutput, error) {
	return &m.DeleteTransitGatewayConnectOutput, nil
}

func TestTransitGateways_GetAll(t *testing.T) {

	t.Parallel()

	now := time.Now()
	gatewayId1 := "gateway1"
	gatewayId2 := "gateway2"
	tgw := TransitGateways{
		Client: mockedTransitGateway{
			DescribeTransitGatewaysOutput: ec2.DescribeTransitGatewaysOutput{
				TransitGateways: []types.TransitGateway{
					{
						TransitGatewayId: aws.String(gatewayId1),
						CreationTime:     aws.Time(now),
						State:            "available",
					},
					{
						TransitGatewayId: aws.String(gatewayId2),
						CreationTime:     aws.Time(now.Add(1)),
						State:            "deleting",
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
