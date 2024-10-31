package resources

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedTransitGatewayVpcAttachment struct {
	DescribeTransitGatewayVpcAttachmentsOutput ec2.DescribeTransitGatewayVpcAttachmentsOutput
	DeleteTransitGatewayVpcAttachmentOutput    ec2.DeleteTransitGatewayVpcAttachmentOutput
}

func (m mockedTransitGatewayVpcAttachment) DescribeTransitGatewayVpcAttachments(ctx context.Context, params *ec2.DescribeTransitGatewayVpcAttachmentsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayVpcAttachmentsOutput, error) {
	return &m.DescribeTransitGatewayVpcAttachmentsOutput, nil
}

func (m mockedTransitGatewayVpcAttachment) DeleteTransitGatewayVpcAttachment(ctx context.Context, params *ec2.DeleteTransitGatewayVpcAttachmentInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayVpcAttachmentOutput, error) {
	return &m.DeleteTransitGatewayVpcAttachmentOutput, nil
}

func TestTransitGatewayVpcAttachments_GetAll(t *testing.T) {

	t.Parallel()

	now := time.Now()
	attachment1 := "attachment1"
	attachment2 := "attachment2"
	tgw := TransitGatewaysVpcAttachment{
		Client: mockedTransitGatewayVpcAttachment{
			DescribeTransitGatewayVpcAttachmentsOutput: ec2.DescribeTransitGatewayVpcAttachmentsOutput{
				TransitGatewayVpcAttachments: []types.TransitGatewayVpcAttachment{
					{
						TransitGatewayAttachmentId: aws.String(attachment1),
						CreationTime:               aws.Time(now),
						State:                      types.TransitGatewayAttachmentStateAvailable,
					},
					{
						TransitGatewayAttachmentId: aws.String(attachment2),
						CreationTime:               aws.Time(now.Add(1 * time.Hour)),
						State:                      types.TransitGatewayAttachmentStateDeleting,
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
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
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
