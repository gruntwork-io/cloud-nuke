package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
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
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

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
						EgressOnlyInternetGatewayId: aws.String(gateway1),
						Tags: []*ec2.Tag{
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
						Tags: []*ec2.Tag{
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
	object.BaseAwsResource.Init(nil)

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{gateway1, gateway2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{gateway2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{gateway1},
		},
		"timeBeforeExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeBefore: aws.Time(now.Add(1)),
				}},
			expected: []string{gateway2},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			names, err := object.getAll(context.Background(), config.Config{
				EgressOnlyInternetGateway: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}

}

func TestEc2EgressOnlyInternetGateway_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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
						EgressOnlyInternetGatewayId: aws.String(gateway1),
						Attachments: []*ec2.InternetGatewayAttachment{
							{
								State: aws.String("testing-state"),
								VpcId: aws.String("test-gateway-vpc"),
							},
						},
					},
					{
						EgressOnlyInternetGatewayId: aws.String(gateway2),
						Attachments: []*ec2.InternetGatewayAttachment{
							{
								State: aws.String("testing-state"),
								VpcId: aws.String("test-gateway-vpc"),
							},
						},
					},
				},
			},
			DeleteEgressOnlyInternetGatewayOutput: ec2.DeleteEgressOnlyInternetGatewayOutput{},
		},
	}

	err := igw.nukeAll([]*string{
		aws.String(gateway1),
		aws.String(gateway2),
	})
	require.NoError(t, err)
}
