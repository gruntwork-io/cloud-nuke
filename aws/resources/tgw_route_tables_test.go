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

type mockedTransitGatewayRouteTable struct {
	ec2iface.EC2API
	DescribeTransitGatewayRouteTablesOutput ec2.DescribeTransitGatewayRouteTablesOutput
	DeleteTransitGatewayRouteTableOutput    ec2.DeleteTransitGatewayRouteTableOutput
}

func (m mockedTransitGatewayRouteTable) DescribeTransitGatewayRouteTablesWithContext(_ awsgo.Context, _ *ec2.DescribeTransitGatewayRouteTablesInput, _ ...request.Option) (*ec2.DescribeTransitGatewayRouteTablesOutput, error) {
	return &m.DescribeTransitGatewayRouteTablesOutput, nil
}

func (m mockedTransitGatewayRouteTable) DeleteTransitGatewayRouteTableWithContext(_ awsgo.Context, _ *ec2.DeleteTransitGatewayRouteTableInput, _ ...request.Option) (*ec2.DeleteTransitGatewayRouteTableOutput, error) {
	return &m.DeleteTransitGatewayRouteTableOutput, nil
}
func TestTransitGatewayRouteTables_GetAll(t *testing.T) {

	t.Parallel()

	now := time.Now()
	tableId1 := "table1"
	tableId2 := "table2"
	tgw := TransitGatewaysRouteTables{
		Client: mockedTransitGatewayRouteTable{
			DescribeTransitGatewayRouteTablesOutput: ec2.DescribeTransitGatewayRouteTablesOutput{
				TransitGatewayRouteTables: []*ec2.TransitGatewayRouteTable{
					{
						TransitGatewayRouteTableId: aws.String(tableId1),
						CreationTime:               aws.Time(now),
						State:                      aws.String("available"),
					},
					{
						TransitGatewayRouteTableId: aws.String(tableId2),
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
			expected:  []string{tableId1},
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
				TransitGatewayRouteTable: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestTransitGatewayRouteTables_NukeAll(t *testing.T) {

	t.Parallel()

	tgw := TransitGatewaysRouteTables{
		Client: mockedTransitGatewayRouteTable{
			DeleteTransitGatewayRouteTableOutput: ec2.DeleteTransitGatewayRouteTableOutput{},
		},
	}

	err := tgw.nukeAll([]*string{aws.String("test-route-table")})
	require.NoError(t, err)
}
