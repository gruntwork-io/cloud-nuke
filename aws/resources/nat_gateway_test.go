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

type mockNatGatewayClient struct {
	DescribeNatGatewaysOutput ec2.DescribeNatGatewaysOutput
	DeleteNatGatewayOutput    ec2.DeleteNatGatewayOutput
}

func (m *mockNatGatewayClient) DescribeNatGateways(ctx context.Context, params *ec2.DescribeNatGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNatGatewaysOutput, error) {
	return &m.DescribeNatGatewaysOutput, nil
}

func (m *mockNatGatewayClient) DeleteNatGateway(ctx context.Context, params *ec2.DeleteNatGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNatGatewayOutput, error) {
	return &m.DeleteNatGatewayOutput, nil
}

func TestListNatGateways(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tests := map[string]struct {
		gateways []types.NatGateway
		config   config.ResourceType
		expected []string
	}{
		"all gateways": {
			gateways: []types.NatGateway{
				{NatGatewayId: aws.String("ngw-1"), CreateTime: aws.Time(now), Tags: []types.Tag{{Key: aws.String("Name"), Value: aws.String("test-1")}}},
				{NatGatewayId: aws.String("ngw-2"), CreateTime: aws.Time(now), Tags: []types.Tag{{Key: aws.String("Name"), Value: aws.String("test-2")}}},
			},
			config:   config.ResourceType{},
			expected: []string{"ngw-1", "ngw-2"},
		},
		"exclude by name": {
			gateways: []types.NatGateway{
				{NatGatewayId: aws.String("ngw-1"), CreateTime: aws.Time(now), Tags: []types.Tag{{Key: aws.String("Name"), Value: aws.String("skip-this")}}},
				{NatGatewayId: aws.String("ngw-2"), CreateTime: aws.Time(now), Tags: []types.Tag{{Key: aws.String("Name"), Value: aws.String("keep-this")}}},
			},
			config: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
				},
			},
			expected: []string{"ngw-2"},
		},
		"skip deleted state": {
			gateways: []types.NatGateway{
				{NatGatewayId: aws.String("ngw-1"), CreateTime: aws.Time(now), State: types.NatGatewayStateDeleted},
				{NatGatewayId: aws.String("ngw-2"), CreateTime: aws.Time(now), State: types.NatGatewayStateAvailable},
			},
			config:   config.ResourceType{},
			expected: []string{"ngw-2"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mock := &mockNatGatewayClient{
				DescribeNatGatewaysOutput: ec2.DescribeNatGatewaysOutput{
					NatGateways: tc.gateways,
				},
			}

			result, err := listNatGateways(context.Background(), mock, resource.Scope{}, tc.config)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(result))
		})
	}
}

func TestDeleteNatGateway(t *testing.T) {
	t.Parallel()

	mock := &mockNatGatewayClient{}
	err := deleteNatGateway(context.Background(), mock, aws.String("ngw-test"))
	require.NoError(t, err)
}
