package resources

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockTransitGatewayVpcAttachmentClient struct {
	DescribeTransitGatewayVpcAttachmentsOutput ec2.DescribeTransitGatewayVpcAttachmentsOutput
	DeleteTransitGatewayVpcAttachmentOutput    ec2.DeleteTransitGatewayVpcAttachmentOutput
}

func (m *mockTransitGatewayVpcAttachmentClient) DescribeTransitGatewayVpcAttachments(ctx context.Context, params *ec2.DescribeTransitGatewayVpcAttachmentsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayVpcAttachmentsOutput, error) {
	return &m.DescribeTransitGatewayVpcAttachmentsOutput, nil
}

func (m *mockTransitGatewayVpcAttachmentClient) DeleteTransitGatewayVpcAttachment(ctx context.Context, params *ec2.DeleteTransitGatewayVpcAttachmentInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayVpcAttachmentOutput, error) {
	return &m.DeleteTransitGatewayVpcAttachmentOutput, nil
}

func TestListTransitGatewayVpcAttachments(t *testing.T) {
	t.Parallel()

	now := time.Now()
	attachment1 := "attachment1"
	attachment2 := "attachment2"

	mock := &mockTransitGatewayVpcAttachmentClient{
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
			names, err := listTransitGatewaysVpcAttachments(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteTransitGatewayVpcAttachments(t *testing.T) {
	t.Parallel()

	mock := &mockTransitGatewayVpcAttachmentClient{
		DeleteTransitGatewayVpcAttachmentOutput: ec2.DeleteTransitGatewayVpcAttachmentOutput{},
	}

	err := deleteTransitGatewaysVpcAttachments(context.Background(), mock, resource.Scope{Region: "us-east-1"}, "transit-gateway-attachment", []*string{aws.String("test-attachment")})
	require.NoError(t, err)
}
