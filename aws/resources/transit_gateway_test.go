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
	"github.com/stretchr/testify/require"
)

// mockTransitGatewaysClient implements TransitGatewaysAPI for testing.
type mockTransitGatewaysClient struct {
	DescribeTransitGatewaysOutput           ec2.DescribeTransitGatewaysOutput
	DeleteTransitGatewayOutput              ec2.DeleteTransitGatewayOutput
	DescribeTransitGatewayAttachmentsOutput ec2.DescribeTransitGatewayAttachmentsOutput
	DeleteTransitGatewayPeeringOutput       ec2.DeleteTransitGatewayPeeringAttachmentOutput
	DeleteTransitGatewayVpcOutput           ec2.DeleteTransitGatewayVpcAttachmentOutput
	DeleteTransitGatewayConnectOutput       ec2.DeleteTransitGatewayConnectOutput
}

func (m *mockTransitGatewaysClient) DescribeTransitGateways(ctx context.Context, params *ec2.DescribeTransitGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewaysOutput, error) {
	return &m.DescribeTransitGatewaysOutput, nil
}

func (m *mockTransitGatewaysClient) DeleteTransitGateway(ctx context.Context, params *ec2.DeleteTransitGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayOutput, error) {
	return &m.DeleteTransitGatewayOutput, nil
}

func (m *mockTransitGatewaysClient) DescribeTransitGatewayAttachments(ctx context.Context, params *ec2.DescribeTransitGatewayAttachmentsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayAttachmentsOutput, error) {
	return &m.DescribeTransitGatewayAttachmentsOutput, nil
}

func (m *mockTransitGatewaysClient) DeleteTransitGatewayPeeringAttachment(ctx context.Context, params *ec2.DeleteTransitGatewayPeeringAttachmentInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayPeeringAttachmentOutput, error) {
	return &m.DeleteTransitGatewayPeeringOutput, nil
}

func (m *mockTransitGatewaysClient) DeleteTransitGatewayVpcAttachment(ctx context.Context, params *ec2.DeleteTransitGatewayVpcAttachmentInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayVpcAttachmentOutput, error) {
	return &m.DeleteTransitGatewayVpcOutput, nil
}

func (m *mockTransitGatewaysClient) DeleteTransitGatewayConnect(ctx context.Context, params *ec2.DeleteTransitGatewayConnectInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayConnectOutput, error) {
	return &m.DeleteTransitGatewayConnectOutput, nil
}

func TestTransitGateways_GetAll(t *testing.T) {
	t.Parallel()

	testID1 := "tgw-001"
	testID2 := "tgw-002"
	now := time.Now()

	mock := &mockTransitGatewaysClient{
		DescribeTransitGatewaysOutput: ec2.DescribeTransitGatewaysOutput{
			TransitGateways: []types.TransitGateway{
				{
					TransitGatewayId: aws.String(testID1),
					CreationTime:     aws.Time(now),
					State:            types.TransitGatewayStateAvailable,
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("test-gateway-1")},
						{Key: aws.String("env"), Value: aws.String("dev")},
					},
				},
				{
					TransitGatewayId: aws.String(testID2),
					CreationTime:     aws.Time(now.Add(1 * time.Hour)),
					State:            types.TransitGatewayStateDeleting, // Should be skipped
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
			expected:  []string{testID1},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("test-gateway-1")}},
				},
			},
			expected: []string{},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				},
			},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := listTransitGateways(context.Background(), mock, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestTransitGateways_Nuke(t *testing.T) {
	t.Parallel()

	mock := &mockTransitGatewaysClient{
		DescribeTransitGatewayAttachmentsOutput: ec2.DescribeTransitGatewayAttachmentsOutput{
			TransitGatewayAttachments: []types.TransitGatewayAttachment{}, // No attachments
		},
		DeleteTransitGatewayOutput: ec2.DeleteTransitGatewayOutput{},
	}

	err := nukeTransitGateway(context.Background(), mock, aws.String("tgw-test"))
	require.NoError(t, err)
}

func TestTransitGateways_NukeWithAttachments(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		attachmentType types.TransitGatewayAttachmentResourceType
	}{
		"vpc_attachment": {
			attachmentType: types.TransitGatewayAttachmentResourceTypeVpc,
		},
		"peering_attachment": {
			attachmentType: types.TransitGatewayAttachmentResourceTypePeering,
		},
		"connect_attachment": {
			attachmentType: types.TransitGatewayAttachmentResourceTypeConnect,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// First call returns attachment, second call returns empty (deleted)
			callCount := 0
			mock := &mockTransitGatewaysClient{
				DeleteTransitGatewayOutput:        ec2.DeleteTransitGatewayOutput{},
				DeleteTransitGatewayPeeringOutput: ec2.DeleteTransitGatewayPeeringAttachmentOutput{},
				DeleteTransitGatewayVpcOutput:     ec2.DeleteTransitGatewayVpcAttachmentOutput{},
				DeleteTransitGatewayConnectOutput: ec2.DeleteTransitGatewayConnectOutput{},
			}

			// Override DescribeTransitGatewayAttachments to simulate deletion
			mockWithCounter := &mockTransitGatewaysClientWithCounter{
				mockTransitGatewaysClient: mock,
				attachmentType:            tc.attachmentType,
				callCount:                 &callCount,
			}

			err := nukeTransitGateway(context.Background(), mockWithCounter, aws.String("tgw-test"))
			require.NoError(t, err)
		})
	}
}

// mockTransitGatewaysClientWithCounter wraps mockTransitGatewaysClient to track calls for wait testing.
type mockTransitGatewaysClientWithCounter struct {
	*mockTransitGatewaysClient
	attachmentType types.TransitGatewayAttachmentResourceType
	callCount      *int
}

func (m *mockTransitGatewaysClientWithCounter) DescribeTransitGatewayAttachments(ctx context.Context, params *ec2.DescribeTransitGatewayAttachmentsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayAttachmentsOutput, error) {
	*m.callCount++
	// First call returns attachment, subsequent calls return empty (simulating deletion)
	if *m.callCount == 1 {
		return &ec2.DescribeTransitGatewayAttachmentsOutput{
			TransitGatewayAttachments: []types.TransitGatewayAttachment{
				{
					TransitGatewayAttachmentId: aws.String("tgw-attach-test"),
					ResourceType:               m.attachmentType,
				},
			},
		}, nil
	}
	return &ec2.DescribeTransitGatewayAttachmentsOutput{}, nil
}

func (m *mockTransitGatewaysClientWithCounter) DescribeTransitGateways(ctx context.Context, params *ec2.DescribeTransitGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTransitGatewaysOutput, error) {
	return m.mockTransitGatewaysClient.DescribeTransitGateways(ctx, params, optFns...)
}

func (m *mockTransitGatewaysClientWithCounter) DeleteTransitGateway(ctx context.Context, params *ec2.DeleteTransitGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayOutput, error) {
	return m.mockTransitGatewaysClient.DeleteTransitGateway(ctx, params, optFns...)
}

func (m *mockTransitGatewaysClientWithCounter) DeleteTransitGatewayPeeringAttachment(ctx context.Context, params *ec2.DeleteTransitGatewayPeeringAttachmentInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayPeeringAttachmentOutput, error) {
	return m.mockTransitGatewaysClient.DeleteTransitGatewayPeeringAttachment(ctx, params, optFns...)
}

func (m *mockTransitGatewaysClientWithCounter) DeleteTransitGatewayVpcAttachment(ctx context.Context, params *ec2.DeleteTransitGatewayVpcAttachmentInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayVpcAttachmentOutput, error) {
	return m.mockTransitGatewaysClient.DeleteTransitGatewayVpcAttachment(ctx, params, optFns...)
}

func (m *mockTransitGatewaysClientWithCounter) DeleteTransitGatewayConnect(ctx context.Context, params *ec2.DeleteTransitGatewayConnectInput, optFns ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayConnectOutput, error) {
	return m.mockTransitGatewaysClient.DeleteTransitGatewayConnect(ctx, params, optFns...)
}
