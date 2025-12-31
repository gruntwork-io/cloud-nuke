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

type mockTransitGatewayPeeringAttachmentClient struct {
	DescribeTransitGatewayPeeringAttachmentsOutput ec2.DescribeTransitGatewayPeeringAttachmentsOutput
	DeleteTransitGatewayPeeringAttachmentOutput    ec2.DeleteTransitGatewayPeeringAttachmentOutput
}

func (m *mockTransitGatewayPeeringAttachmentClient) DescribeTransitGatewayPeeringAttachments(_ context.Context, _ *ec2.DescribeTransitGatewayPeeringAttachmentsInput, _ ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayPeeringAttachmentsOutput, error) {
	return &m.DescribeTransitGatewayPeeringAttachmentsOutput, nil
}

func (m *mockTransitGatewayPeeringAttachmentClient) DeleteTransitGatewayPeeringAttachment(_ context.Context, _ *ec2.DeleteTransitGatewayPeeringAttachmentInput, _ ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayPeeringAttachmentOutput, error) {
	return &m.DeleteTransitGatewayPeeringAttachmentOutput, nil
}

func TestListTransitGatewayPeeringAttachments(t *testing.T) {
	t.Parallel()

	now := time.Now()
	attachment1 := "attachement1"
	attachment2 := "attachement2"

	mock := &mockTransitGatewayPeeringAttachmentClient{
		DescribeTransitGatewayPeeringAttachmentsOutput: ec2.DescribeTransitGatewayPeeringAttachmentsOutput{
			TransitGatewayPeeringAttachments: []types.TransitGatewayPeeringAttachment{
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
			ids, err := listTransitGatewayPeeringAttachments(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestDeleteTransitGatewayPeeringAttachment(t *testing.T) {
	t.Parallel()

	mock := &mockTransitGatewayPeeringAttachmentClient{}
	err := deleteTransitGatewayPeeringAttachment(context.Background(), mock, aws.String("test-attachment"))
	require.NoError(t, err)
}
