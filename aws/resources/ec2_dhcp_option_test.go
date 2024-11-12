package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedEC2DhcpOption struct {
	EC2DhcpOptionAPI
	AssociateDhcpOptionsOutput ec2.AssociateDhcpOptionsOutput
	DescribeDhcpOptionsOutput  ec2.DescribeDhcpOptionsOutput
	DescribeVpcsOutput         ec2.DescribeVpcsOutput
	DeleteDhcpOptionsOutput    ec2.DeleteDhcpOptionsOutput
}

func (m mockedEC2DhcpOption) AssociateDhcpOptions(ctx context.Context, params *ec2.AssociateDhcpOptionsInput, optFns ...func(*ec2.Options)) (*ec2.AssociateDhcpOptionsOutput, error) {
	return &m.AssociateDhcpOptionsOutput, nil
}

func (m mockedEC2DhcpOption) DescribeDhcpOptions(ctx context.Context, params *ec2.DescribeDhcpOptionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeDhcpOptionsOutput, error) {
	return &m.DescribeDhcpOptionsOutput, nil
}

func (m mockedEC2DhcpOption) DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	return &m.DescribeVpcsOutput, nil
}

func (m mockedEC2DhcpOption) DeleteDhcpOptions(ctx context.Context, params *ec2.DeleteDhcpOptionsInput, optFns ...func(*ec2.Options)) (*ec2.DeleteDhcpOptionsOutput, error) {
	return &m.DeleteDhcpOptionsOutput, nil
}

func TestEC2DhcpOption_GetAll(t *testing.T) {
	t.Parallel()

	testId1 := "test-id-1"
	h := EC2DhcpOption{
		Client: mockedEC2DhcpOption{
			DescribeDhcpOptionsOutput: ec2.DescribeDhcpOptionsOutput{
				DhcpOptions: []types.DhcpOptions{
					{
						DhcpOptionsId: aws.String(testId1),
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
			expected:  []string{testId1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := h.getAll(context.Background(), config.Config{
				EC2DedicatedHosts: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestEC2DhcpOption_NukeAll(t *testing.T) {
	t.Parallel()
	h := EC2DhcpOption{
		Client: mockedEC2DhcpOption{
			DeleteDhcpOptionsOutput: ec2.DeleteDhcpOptionsOutput{},
		},
	}

	err := h.nukeAll([]*string{aws.String("test-id-1"), aws.String("test-id-2")})
	require.NoError(t, err)
}
