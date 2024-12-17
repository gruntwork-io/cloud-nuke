package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedNatGateway struct {
	NatGatewaysAPI
	DeleteNatGatewayOutput    ec2.DeleteNatGatewayOutput
	DescribeNatGatewaysOutput ec2.DescribeNatGatewaysOutput
	DescribeNatGatewaysError  error
}

func (m mockedNatGateway) DeleteNatGateway(ctx context.Context, params *ec2.DeleteNatGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNatGatewayOutput, error) {
	return &m.DeleteNatGatewayOutput, nil
}

func (m mockedNatGateway) DescribeNatGateways(ctx context.Context, params *ec2.DescribeNatGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNatGatewaysOutput, error) {
	return &m.DescribeNatGatewaysOutput, m.DescribeNatGatewaysError
}

func TestNatGateway_GetAll(t *testing.T) {

	t.Parallel()

	testId1 := "test-nat-gateway-id1"
	testId2 := "test-nat-gateway-id2"
	testName1 := "test-nat-gateway-1"
	testName2 := "test-nat-gateway-2"
	now := time.Now()
	ng := NatGateways{
		Client: mockedNatGateway{
			DescribeNatGatewaysOutput: ec2.DescribeNatGatewaysOutput{
				NatGateways: []types.NatGateway{
					{
						NatGatewayId: aws.String(testId1),
						Tags: []types.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName1),
							},
						},
						CreateTime: aws.Time(now),
					},
					{
						NatGatewayId: aws.String(testId2),
						Tags: []types.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName2),
							},
						},
						CreateTime: aws.Time(now.Add(1)),
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
			expected:  []string{testId1, testId2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testId2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testId1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := ng.getAll(context.Background(), config.Config{
				NatGateway: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestNatGateway_NukeAll(t *testing.T) {

	t.Parallel()

	ngw := NatGateways{
		Client: mockedNatGateway{
			DeleteNatGatewayOutput: ec2.DeleteNatGatewayOutput{},
			DescribeNatGatewaysError: &smithy.GenericAPIError{
				Code: "NatGatewayNotFound",
			},
		},
	}

	err := ngw.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
