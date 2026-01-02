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

// mockTransitGatewayRouteTablesClient implements TransitGatewaysRouteTablesAPI for testing.
type mockTransitGatewayRouteTablesClient struct {
	DescribeTransitGatewayRouteTablesOutput ec2.DescribeTransitGatewayRouteTablesOutput
	DeleteTransitGatewayRouteTableOutput    ec2.DeleteTransitGatewayRouteTableOutput
}

func (m *mockTransitGatewayRouteTablesClient) DescribeTransitGatewayRouteTables(_ context.Context, _ *ec2.DescribeTransitGatewayRouteTablesInput, _ ...func(*ec2.Options)) (*ec2.DescribeTransitGatewayRouteTablesOutput, error) {
	return &m.DescribeTransitGatewayRouteTablesOutput, nil
}

func (m *mockTransitGatewayRouteTablesClient) DeleteTransitGatewayRouteTable(_ context.Context, _ *ec2.DeleteTransitGatewayRouteTableInput, _ ...func(*ec2.Options)) (*ec2.DeleteTransitGatewayRouteTableOutput, error) {
	return &m.DeleteTransitGatewayRouteTableOutput, nil
}

func TestTransitGatewayRouteTables_List(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tableId1 := "tgw-rtb-111111"
	tableId2 := "tgw-rtb-222222"
	tableId3 := "tgw-rtb-333333"

	mock := &mockTransitGatewayRouteTablesClient{
		DescribeTransitGatewayRouteTablesOutput: ec2.DescribeTransitGatewayRouteTablesOutput{
			TransitGatewayRouteTables: []types.TransitGatewayRouteTable{
				{
					TransitGatewayRouteTableId: aws.String(tableId1),
					CreationTime:               aws.Time(now),
					State:                      types.TransitGatewayRouteTableStateAvailable,
				},
				{
					TransitGatewayRouteTableId: aws.String(tableId2),
					CreationTime:               aws.Time(now.Add(1 * time.Hour)),
					State:                      types.TransitGatewayRouteTableStateDeleting,
				},
				{
					TransitGatewayRouteTableId: aws.String(tableId3),
					CreationTime:               aws.Time(now.Add(-2 * time.Hour)),
					State:                      types.TransitGatewayRouteTableStateAvailable,
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
			expected:  []string{tableId1, tableId3},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				},
			},
			expected: []string{tableId3},
		},
		"timeBeforeExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeBefore: aws.Time(now.Add(-1 * time.Hour)),
				},
			},
			expected: []string{tableId1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := listTransitGatewayRouteTables(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestTransitGatewayRouteTables_Delete(t *testing.T) {
	t.Parallel()

	mock := &mockTransitGatewayRouteTablesClient{}
	err := deleteTransitGatewayRouteTable(context.Background(), mock, aws.String("tgw-rtb-test"))
	require.NoError(t, err)
}
