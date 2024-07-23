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
