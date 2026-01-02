package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockedEC2DhcpOption struct {
	EC2DhcpOptionAPI
	DescribeDhcpOptionsOutput ec2.DescribeDhcpOptionsOutput
	DescribeVpcsOutput        ec2.DescribeVpcsOutput
	AssociateDhcpOptionsErr   error
	DeleteDhcpOptionsErr      error
}

func (m mockedEC2DhcpOption) AssociateDhcpOptions(ctx context.Context, params *ec2.AssociateDhcpOptionsInput, optFns ...func(*ec2.Options)) (*ec2.AssociateDhcpOptionsOutput, error) {
	return &ec2.AssociateDhcpOptionsOutput{}, m.AssociateDhcpOptionsErr
}

func (m mockedEC2DhcpOption) DescribeDhcpOptions(ctx context.Context, params *ec2.DescribeDhcpOptionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeDhcpOptionsOutput, error) {
	return &m.DescribeDhcpOptionsOutput, nil
}

func (m mockedEC2DhcpOption) DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	return &m.DescribeVpcsOutput, nil
}

func (m mockedEC2DhcpOption) DeleteDhcpOptions(ctx context.Context, params *ec2.DeleteDhcpOptionsInput, optFns ...func(*ec2.Options)) (*ec2.DeleteDhcpOptionsOutput, error) {
	return &ec2.DeleteDhcpOptionsOutput{}, m.DeleteDhcpOptionsErr
}

func TestEC2DhcpOption_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		dhcpOptions []types.DhcpOptions
		vpcs        []types.Vpc
		expectedIDs []string
	}{
		{
			name: "single DHCP option without VPC",
			dhcpOptions: []types.DhcpOptions{
				{DhcpOptionsId: aws.String("dopt-123")},
			},
			vpcs:        []types.Vpc{},
			expectedIDs: []string{"dopt-123"},
		},
		{
			name: "DHCP option with non-default VPC",
			dhcpOptions: []types.DhcpOptions{
				{DhcpOptionsId: aws.String("dopt-456")},
			},
			vpcs: []types.Vpc{
				{VpcId: aws.String("vpc-abc"), IsDefault: aws.Bool(false)},
			},
			expectedIDs: []string{"dopt-456"},
		},
		{
			name: "skip DHCP option attached to default VPC",
			dhcpOptions: []types.DhcpOptions{
				{DhcpOptionsId: aws.String("dopt-789")},
			},
			vpcs: []types.Vpc{
				{VpcId: aws.String("vpc-default"), IsDefault: aws.Bool(true)},
			},
			expectedIDs: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := mockedEC2DhcpOption{
				DescribeDhcpOptionsOutput: ec2.DescribeDhcpOptionsOutput{
					DhcpOptions: tc.dhcpOptions,
				},
				DescribeVpcsOutput: ec2.DescribeVpcsOutput{
					Vpcs: tc.vpcs,
				},
			}

			ids, err := listEC2DhcpOptions(context.Background(), client, resource.Scope{Region: "us-east-1"}, config.ResourceType{})
			require.NoError(t, err)
			require.Equal(t, tc.expectedIDs, aws.ToStringSlice(ids))
		})
	}
}

func TestEC2DhcpOption_Nuke(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ids       []string
		vpcs      []types.Vpc
		expectErr bool
	}{
		{
			name: "delete DHCP option without VPC association",
			ids:  []string{"dopt-123"},
			vpcs: []types.Vpc{},
		},
		{
			name: "delete DHCP option with VPC association",
			ids:  []string{"dopt-456"},
			vpcs: []types.Vpc{
				{VpcId: aws.String("vpc-abc"), IsDefault: aws.Bool(false)},
			},
		},
		{
			name: "delete multiple DHCP options",
			ids:  []string{"dopt-1", "dopt-2"},
			vpcs: []types.Vpc{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := mockedEC2DhcpOption{
				DescribeVpcsOutput: ec2.DescribeVpcsOutput{
					Vpcs: tc.vpcs,
				},
			}
			nuker := resource.MultiStepDeleter(
				disassociateDhcpOption,
				deleteDhcpOption,
			)

			results := nuker(context.Background(), client, resource.Scope{Region: "us-east-1"}, "ec2-dhcp-option", aws.StringSlice(tc.ids))
			for _, result := range results {
				if tc.expectErr {
					require.Error(t, result.Error)
				} else {
					require.NoError(t, result.Error)
				}
			}
		})
	}
}
