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

type mockedTransitGatewayVpcAttachment struct {
	ec2iface.EC2API
	DescribeTransitGatewayVpcAttachmentsOutput ec2.DescribeTransitGatewayVpcAttachmentsOutput
	DeleteTransitGatewayVpcAttachmentOutput    ec2.DeleteTransitGatewayVpcAttachmentOutput
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
