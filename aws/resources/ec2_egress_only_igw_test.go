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

type mockedEgressOnlyIgw struct {
	BaseAwsResource
	EgressOnlyIGAPI
	DescribeEgressOnlyInternetGatewaysOutput ec2.DescribeEgressOnlyInternetGatewaysOutput
	DeleteEgressOnlyInternetGatewayOutput    ec2.DeleteEgressOnlyInternetGatewayOutput
}

func (m mockedEgressOnlyIgw) DescribeEgressOnlyInternetGateways(ctx context.Context, params *ec2.DescribeEgressOnlyInternetGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeEgressOnlyInternetGatewaysOutput, error) {
	return &m.DescribeEgressOnlyInternetGatewaysOutput, nil
}

func (m mockedEgressOnlyIgw) DeleteEgressOnlyInternetGateway(ctx context.Context, params *ec2.DeleteEgressOnlyInternetGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteEgressOnlyInternetGatewayOutput, error) {
	return &m.DeleteEgressOnlyInternetGatewayOutput, nil
}

func TestEgressOnlyInternetGateway_GetAll(t *testing.T) {

	t.Parallel()

	// Set excludeFirstSeenTag to false for testing
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	var (
		now      = time.Now()
		gateway1 = "igw-0b44cfa6103932e1d001"
		gateway2 = "igw-0b44cfa6103932e1d002"

		testName1 = "cloud-nuke-igw-001"
		testName2 = "cloud-nuke-igw-002"
	)
	object := EgressOnlyInternetGateway{
		Client: mockedEgressOnlyIgw{
			DescribeEgressOnlyInternetGatewaysOutput: ec2.DescribeEgressOnlyInternetGatewaysOutput{
				EgressOnlyInternetGateways: []types.EgressOnlyInternetGateway{
					{
						EgressOnlyInternetGatewayId: aws.String(gateway1),
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
						EgressOnlyInternetGatewayId: aws.String(gateway2),
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
			expected:  []string{gateway1, gateway2},
		},
		"nameExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{gateway2},
		},
		"timeAfterExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{gateway1},
		},
		"timeBeforeExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeBefore: aws.Time(now.Add(1)),
				}},
			expected: []string{gateway2},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := object.getAll(tc.ctx, config.Config{
				EgressOnlyInternetGateway: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}

}

func TestEc2EgressOnlyInternetGateway_NukeAll(t *testing.T) {

	t.Parallel()

	var (
		gateway1 = "igw-0b44cfa6103932e1d001"
		gateway2 = "igw-0b44cfa6103932e1d002"
	)

	igw := EgressOnlyInternetGateway{
		Client: mockedEgressOnlyIgw{
			DescribeEgressOnlyInternetGatewaysOutput: ec2.DescribeEgressOnlyInternetGatewaysOutput{
				EgressOnlyInternetGateways: []types.EgressOnlyInternetGateway{
					{
						EgressOnlyInternetGatewayId: aws.String(gateway1),
						Attachments: []types.InternetGatewayAttachment{
							{
								State: "testing-state",
								VpcId: aws.String("test-gateway-vpc"),
							},
						},
					},
					{
						EgressOnlyInternetGatewayId: aws.String(gateway2),
						Attachments: []types.InternetGatewayAttachment{
							{
								State: "testing-state",
								VpcId: aws.String("test-gateway-vpc"),
							},
						},
					},
				},
			},
			DeleteEgressOnlyInternetGatewayOutput: ec2.DeleteEgressOnlyInternetGatewayOutput{},
		},
	}
	igw.Nukables = map[string]error{
		gateway1: nil,
	}

	err := igw.nukeAll([]*string{
		aws.String(gateway1),
		aws.String(gateway2),
	})
	require.NoError(t, err)
}
