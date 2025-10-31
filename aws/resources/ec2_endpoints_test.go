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
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedEc2VpcEndpoints struct {
	EC2EndpointsAPI
	DescribeVpcEndpointsOutput ec2.DescribeVpcEndpointsOutput
	DeleteVpcEndpointsOutput   ec2.DeleteVpcEndpointsOutput
}

func (m mockedEc2VpcEndpoints) DescribeVpcEndpoints(ctx context.Context, params *ec2.DescribeVpcEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointsOutput, error) {
	return &m.DescribeVpcEndpointsOutput, nil
}

func (m mockedEc2VpcEndpoints) DeleteVpcEndpoints(ctx context.Context, params *ec2.DeleteVpcEndpointsInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcEndpointsOutput, error) {
	return &m.DeleteVpcEndpointsOutput, nil
}

func TestVcpEndpoint_GetAll(t *testing.T) {
	t.Parallel()

	// Set excludeFirstSeenTag to false for testing
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	var (
		now       = time.Now()
		endpoint1 = "vpce-0b201b2dcd4f77a2f001"
		endpoint2 = "vpce-0b201b2dcd4f77a2f002"

		testName1 = "cloud-nuke-igw-001"
		testName2 = "cloud-nuke-igw-002"
	)
	vpcEndpoint := EC2Endpoints{
		Client: mockedEc2VpcEndpoints{
			DescribeVpcEndpointsOutput: ec2.DescribeVpcEndpointsOutput{
				VpcEndpoints: []types.VpcEndpoint{
					{
						VpcEndpointId: aws.String(endpoint1),
						Tags: []types.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName1),
							}, {
								Key:   aws.String(util.FirstSeenTagKey),
								Value: aws.String(util.FormatTimestamp(now)),
							},
						},
					},
					{
						VpcEndpointId: aws.String(endpoint2),
						Tags: []types.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName2),
							}, {
								Key:   aws.String(util.FirstSeenTagKey),
								Value: aws.String(util.FormatTimestamp(now.Add(1 * time.Hour))),
							},
						},
					},
				},
			},
		},
	}

	tests := map[string]struct {
		ctx       context.Context
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			ctx:       ctx,
			configObj: config.ResourceType{},
			expected:  []string{endpoint1, endpoint2},
		},
		"nameExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{endpoint2},
		},
		"timeAfterExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{endpoint1},
		},
		"timeBeforeExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeBefore: aws.Time(now.Add(1)),
				}},
			expected: []string{endpoint2},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := vpcEndpoint.getAll(tc.ctx, config.Config{
				EC2Endpoint: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestEc2Endpoints_NukeAll(t *testing.T) {
	t.Parallel()
	var (
		endpoint1 = "vpce-0b201b2dcd4f77a2f001"
		endpoint2 = "vpce-0b201b2dcd4f77a2f002"
	)

	igw := EC2Endpoints{
		Client: mockedEc2VpcEndpoints{
			DescribeVpcEndpointsOutput: ec2.DescribeVpcEndpointsOutput{
				VpcEndpoints: []types.VpcEndpoint{
					{
						VpcEndpointId: aws.String(endpoint1),
					},
					{
						VpcEndpointId: aws.String(endpoint2),
					},
				},
			},
			DeleteVpcEndpointsOutput: ec2.DeleteVpcEndpointsOutput{},
		},
	}
	igw.Nukables = map[string]error{
		endpoint1: nil,
		endpoint2: nil,
	}

	err := igw.nukeAll([]*string{
		aws.String(endpoint1),
		aws.String(endpoint2),
	})
	require.NoError(t, err)
}
