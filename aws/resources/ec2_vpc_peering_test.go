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

type mockVPCPeeringClient struct {
	DescribeVpcPeeringConnectionsOutput ec2.DescribeVpcPeeringConnectionsOutput
	DeleteVpcPeeringConnectionOutput    ec2.DeleteVpcPeeringConnectionOutput

	DeletedIDs []string
}

func (m *mockVPCPeeringClient) DescribeVpcPeeringConnections(ctx context.Context, params *ec2.DescribeVpcPeeringConnectionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcPeeringConnectionsOutput, error) {
	return &m.DescribeVpcPeeringConnectionsOutput, nil
}

func (m *mockVPCPeeringClient) DeleteVpcPeeringConnection(ctx context.Context, params *ec2.DeleteVpcPeeringConnectionInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcPeeringConnectionOutput, error) {
	m.DeletedIDs = append(m.DeletedIDs, aws.ToString(params.VpcPeeringConnectionId))
	return &m.DeleteVpcPeeringConnectionOutput, nil
}

func (m *mockVPCPeeringClient) CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error) {
	return &ec2.CreateTagsOutput{}, nil
}

func TestListVPCPeeringConnections(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := map[string]struct {
		connections []types.VpcPeeringConnection
		config      config.ResourceType
		expected    []string
	}{
		"includes active connections": {
			connections: []types.VpcPeeringConnection{
				{
					VpcPeeringConnectionId: aws.String("pcx-1"),
					Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodeActive},
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("test-peering")},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
			},
			config:   config.ResourceType{},
			expected: []string{"pcx-1"},
		},
		"includes pending-acceptance connections": {
			connections: []types.VpcPeeringConnection{
				{
					VpcPeeringConnectionId: aws.String("pcx-pending"),
					Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodePendingAcceptance},
					Tags: []types.Tag{
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
			},
			config:   config.ResourceType{},
			expected: []string{"pcx-pending"},
		},
		"skips terminal states": {
			connections: []types.VpcPeeringConnection{
				{
					VpcPeeringConnectionId: aws.String("pcx-deleted"),
					Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodeDeleted},
					Tags: []types.Tag{
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					VpcPeeringConnectionId: aws.String("pcx-deleting"),
					Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodeDeleting},
					Tags: []types.Tag{
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					VpcPeeringConnectionId: aws.String("pcx-rejected"),
					Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodeRejected},
					Tags: []types.Tag{
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					VpcPeeringConnectionId: aws.String("pcx-failed"),
					Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodeFailed},
					Tags: []types.Tag{
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					VpcPeeringConnectionId: aws.String("pcx-expired"),
					Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodeExpired},
					Tags: []types.Tag{
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					VpcPeeringConnectionId: aws.String("pcx-active"),
					Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodeActive},
					Tags: []types.Tag{
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
			},
			config:   config.ResourceType{},
			expected: []string{"pcx-active"},
		},
		"handles nil status gracefully": {
			connections: []types.VpcPeeringConnection{
				{
					VpcPeeringConnectionId: aws.String("pcx-nil"),
					Status:                 nil,
					Tags: []types.Tag{
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
			},
			config:   config.ResourceType{},
			expected: []string{"pcx-nil"},
		},
		"exclude by name": {
			connections: []types.VpcPeeringConnection{
				{
					VpcPeeringConnectionId: aws.String("pcx-1"),
					Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodeActive},
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("skip-this")},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					VpcPeeringConnectionId: aws.String("pcx-2"),
					Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodeActive},
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("keep-this")},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
			},
			config: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
				},
			},
			expected: []string{"pcx-2"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mock := &mockVPCPeeringClient{
				DescribeVpcPeeringConnectionsOutput: ec2.DescribeVpcPeeringConnectionsOutput{
					VpcPeeringConnections: tc.connections,
				},
			}

			ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
			result, err := listVPCPeeringConnections(ctx, mock, resource.Scope{}, tc.config)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(result))
		})
	}
}

func TestDeleteVpcPeeringConnection(t *testing.T) {
	t.Parallel()

	mock := &mockVPCPeeringClient{}
	err := deleteVpcPeeringConnection(context.Background(), mock, aws.String("pcx-test"))
	require.NoError(t, err)
	require.Equal(t, []string{"pcx-test"}, mock.DeletedIDs)
}
