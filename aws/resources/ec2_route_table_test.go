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

type mockRouteTableClient struct {
	DescribeRouteTablesOutput    ec2.DescribeRouteTablesOutput
	DisassociateRouteTableOutput ec2.DisassociateRouteTableOutput
	DeleteRouteTableOutput       ec2.DeleteRouteTableOutput
	DescribeVpcsOutput           ec2.DescribeVpcsOutput

	DisassociatedIDs []string
	DeletedIDs       []string
}

func (m *mockRouteTableClient) DescribeRouteTables(ctx context.Context, params *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error) {
	return &m.DescribeRouteTablesOutput, nil
}

func (m *mockRouteTableClient) DisassociateRouteTable(ctx context.Context, params *ec2.DisassociateRouteTableInput, optFns ...func(*ec2.Options)) (*ec2.DisassociateRouteTableOutput, error) {
	m.DisassociatedIDs = append(m.DisassociatedIDs, aws.ToString(params.AssociationId))
	return &m.DisassociateRouteTableOutput, nil
}

func (m *mockRouteTableClient) DeleteRouteTable(ctx context.Context, params *ec2.DeleteRouteTableInput, optFns ...func(*ec2.Options)) (*ec2.DeleteRouteTableOutput, error) {
	m.DeletedIDs = append(m.DeletedIDs, aws.ToString(params.RouteTableId))
	return &m.DeleteRouteTableOutput, nil
}

func (m *mockRouteTableClient) DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	return &m.DescribeVpcsOutput, nil
}

func (m *mockRouteTableClient) CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error) {
	return &ec2.CreateTagsOutput{}, nil
}

func TestListRouteTables(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := map[string]struct {
		routeTables []types.RouteTable
		config      config.ResourceType
		expected    []string
	}{
		"includes non-main route tables": {
			routeTables: []types.RouteTable{
				{
					RouteTableId: aws.String("rtb-1"),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("custom-rt")},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
					Associations: []types.RouteTableAssociation{
						{Main: aws.Bool(false), RouteTableAssociationId: aws.String("rtbassoc-1")},
					},
				},
			},
			config:   config.ResourceType{},
			expected: []string{"rtb-1"},
		},
		"skips main route tables": {
			routeTables: []types.RouteTable{
				{
					RouteTableId: aws.String("rtb-main"),
					Tags: []types.Tag{
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
					Associations: []types.RouteTableAssociation{
						{Main: aws.Bool(true), RouteTableAssociationId: aws.String("rtbassoc-main")},
					},
				},
				{
					RouteTableId: aws.String("rtb-custom"),
					Tags: []types.Tag{
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
			},
			config:   config.ResourceType{},
			expected: []string{"rtb-custom"},
		},
		"includes orphaned route tables with no associations": {
			routeTables: []types.RouteTable{
				{
					RouteTableId: aws.String("rtb-orphan"),
					Tags: []types.Tag{
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
					// No associations at all
				},
			},
			config:   config.ResourceType{},
			expected: []string{"rtb-orphan"},
		},
		"exclude by name": {
			routeTables: []types.RouteTable{
				{
					RouteTableId: aws.String("rtb-1"),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("skip-this")},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					RouteTableId: aws.String("rtb-2"),
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
			expected: []string{"rtb-2"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mock := &mockRouteTableClient{
				DescribeRouteTablesOutput: ec2.DescribeRouteTablesOutput{
					RouteTables: tc.routeTables,
				},
			}

			ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
			result, err := listRouteTables(ctx, mock, resource.Scope{}, tc.config, false)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(result))
		})
	}
}

func TestDisassociateRouteTableSubnets(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		routeTable            types.RouteTable
		expectedDisassociated []string
	}{
		"disassociates non-main associations": {
			routeTable: types.RouteTable{
				RouteTableId: aws.String("rtb-1"),
				Associations: []types.RouteTableAssociation{
					{Main: aws.Bool(false), RouteTableAssociationId: aws.String("rtbassoc-1")},
					{Main: aws.Bool(false), RouteTableAssociationId: aws.String("rtbassoc-2")},
				},
			},
			expectedDisassociated: []string{"rtbassoc-1", "rtbassoc-2"},
		},
		"skips main associations": {
			routeTable: types.RouteTable{
				RouteTableId: aws.String("rtb-1"),
				Associations: []types.RouteTableAssociation{
					{Main: aws.Bool(true), RouteTableAssociationId: aws.String("rtbassoc-main")},
					{Main: aws.Bool(false), RouteTableAssociationId: aws.String("rtbassoc-sub")},
				},
			},
			expectedDisassociated: []string{"rtbassoc-sub"},
		},
		"handles no associations": {
			routeTable: types.RouteTable{
				RouteTableId: aws.String("rtb-1"),
			},
			expectedDisassociated: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mock := &mockRouteTableClient{
				DescribeRouteTablesOutput: ec2.DescribeRouteTablesOutput{
					RouteTables: []types.RouteTable{tc.routeTable},
				},
			}

			err := disassociateRouteTableSubnets(context.Background(), mock, tc.routeTable.RouteTableId)
			require.NoError(t, err)
			require.Equal(t, tc.expectedDisassociated, mock.DisassociatedIDs)
		})
	}
}

func TestDeleteRouteTable(t *testing.T) {
	t.Parallel()

	mock := &mockRouteTableClient{}
	err := deleteRouteTable(context.Background(), mock, aws.String("rtb-test"))
	require.NoError(t, err)
	require.Equal(t, []string{"rtb-test"}, mock.DeletedIDs)
}

func TestIsMainRouteTable(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		rt       types.RouteTable
		expected bool
	}{
		"main route table": {
			rt: types.RouteTable{
				Associations: []types.RouteTableAssociation{
					{Main: aws.Bool(true)},
				},
			},
			expected: true,
		},
		"non-main route table": {
			rt: types.RouteTable{
				Associations: []types.RouteTableAssociation{
					{Main: aws.Bool(false)},
				},
			},
			expected: false,
		},
		"no associations (orphaned)": {
			rt:       types.RouteTable{},
			expected: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, isMainRouteTable(tc.rt))
		})
	}
}
