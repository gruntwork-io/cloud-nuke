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
	DescribeTransitGatewayAttachmentsOutput    ec2.DescribeTransitGatewayAttachmentsOutput
	DeleteTransitGatewayPeeringAttachmentOutput    ec2.DeleteTransitGatewayPeeringAttachmentOutput
}

func (m mockedTransitGateway) DescribeTransitGatewaysWithContext(_ awsgo.Context, _ *ec2.DescribeTransitGatewaysInput, _ ...request.Option) (*ec2.DescribeTransitGatewaysOutput, error) {
	return &m.DescribeTransitGatewaysOutput, nil
}

func (m mockedTransitGateway) DeleteTransitGatewayWithContext(_ awsgo.Context, _ *ec2.DeleteTransitGatewayInput, _ ...request.Option) (*ec2.DeleteTransitGatewayOutput, error) {
	return &m.DeleteTransitGatewayOutput, nil
}

func (m mockedTransitGateway) DescribeTransitGatewayAttachmentsWithContext(awsgo.Context, *ec2.DescribeTransitGatewayAttachmentsInput, ...request.Option) (*ec2.DescribeTransitGatewayAttachmentsOutput, error){
	return &m.DescribeTransitGatewayAttachmentsOutput, nil
}

func (m mockedTransitGateway) DeleteTransitGatewayPeeringAttachmentWithContext(aws.Context, *ec2.DeleteTransitGatewayPeeringAttachmentInput, ...request.Option) (*ec2.DeleteTransitGatewayPeeringAttachmentOutput, error){
	return &m.DeleteTransitGatewayPeeringAttachmentOutput, nil
}
func (m mockedTransitGateway) WaitUntilTransitGatewayAttachmentDeleted(*string, string) error {
	return nil
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
