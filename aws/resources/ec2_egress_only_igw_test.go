package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedEgressOnlyIgw struct {
	BaseAwsResource
	ec2iface.EC2API
	DescribeEgressOnlyInternetGatewaysOutput ec2.DescribeEgressOnlyInternetGatewaysOutput
	DeleteEgressOnlyInternetGatewayOutput    ec2.DeleteEgressOnlyInternetGatewayOutput
}

func (m mockedEgressOnlyIgw) DescribeEgressOnlyInternetGateways(_ *ec2.DescribeEgressOnlyInternetGatewaysInput) (*ec2.DescribeEgressOnlyInternetGatewaysOutput, error) {
	return &m.DescribeEgressOnlyInternetGatewaysOutput, nil
}

func (m mockedEgressOnlyIgw) DeleteEgressOnlyInternetGateway(_ *ec2.DeleteEgressOnlyInternetGatewayInput) (*ec2.DeleteEgressOnlyInternetGatewayOutput, error) {
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
				EgressOnlyInternetGateways: []*ec2.EgressOnlyInternetGateway{
					{
						EgressOnlyInternetGatewayId: awsgo.String(gateway1),
						Tags: []*ec2.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName1),
							}, {
								Key:   awsgo.String(util.FirstSeenTagKey),
								Value: awsgo.String(util.FormatTimestamp(now)),
							},
						},
					},
					{
						EgressOnlyInternetGatewayId: awsgo.String(gateway2),
						Tags: []*ec2.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName2),
							}, {
								Key:   awsgo.String(util.FirstSeenTagKey),
								Value: awsgo.String(util.FormatTimestamp(now.Add(1 * time.Hour))),
							},
						},
					},
				},
			},
		},
	}
	object.BaseAwsResource.Init(nil)

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
					TimeAfter: awsgo.Time(now),
				}},
			expected: []string{gateway1},
		},
		"timeBeforeExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeBefore: awsgo.Time(now.Add(1)),
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
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
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
		BaseAwsResource: BaseAwsResource{
			Nukables: map[string]error{
				gateway1: nil,
			},
		},
		Client: mockedEgressOnlyIgw{
			DescribeEgressOnlyInternetGatewaysOutput: ec2.DescribeEgressOnlyInternetGatewaysOutput{
				EgressOnlyInternetGateways: []*ec2.EgressOnlyInternetGateway{
					{
						EgressOnlyInternetGatewayId: awsgo.String(gateway1),
						Attachments: []*ec2.InternetGatewayAttachment{
							{
								State: awsgo.String("testing-state"),
								VpcId: awsgo.String("test-gateway-vpc"),
							},
						},
					},
					{
						EgressOnlyInternetGatewayId: awsgo.String(gateway2),
						Attachments: []*ec2.InternetGatewayAttachment{
							{
								State: awsgo.String("testing-state"),
								VpcId: awsgo.String("test-gateway-vpc"),
							},
						},
					},
				},
			},
			DeleteEgressOnlyInternetGatewayOutput: ec2.DeleteEgressOnlyInternetGatewayOutput{},
		},
	}

	err := igw.nukeAll([]*string{
		awsgo.String(gateway1),
		awsgo.String(gateway2),
	})
	require.NoError(t, err)
}
