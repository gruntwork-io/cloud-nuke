package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedInternetGateway struct {
	BaseAwsResource
	ec2iface.EC2API
	DescribeInternetGatewaysOutput ec2.DescribeInternetGatewaysOutput
	DetachInternetGatewayOutput    ec2.DetachInternetGatewayOutput
	DeleteInternetGatewayOutput    ec2.DeleteInternetGatewayOutput
}

func (m mockedInternetGateway) DescribeInternetGatewaysWithContext(_ awsgo.Context, _ *ec2.DescribeInternetGatewaysInput, _ ...request.Option) (*ec2.DescribeInternetGatewaysOutput, error) {
	return &m.DescribeInternetGatewaysOutput, nil
}

func (m mockedInternetGateway) DetachInternetGatewayWithContext(_ awsgo.Context, _ *ec2.DetachInternetGatewayInput, _ ...request.Option) (*ec2.DetachInternetGatewayOutput, error) {
	return &m.DetachInternetGatewayOutput, nil
}

func (m mockedInternetGateway) DeleteInternetGatewayWithContext(_ awsgo.Context, _ *ec2.DeleteInternetGatewayInput, _ ...request.Option) (*ec2.DeleteInternetGatewayOutput, error) {
	return &m.DeleteInternetGatewayOutput, nil
}

func TestEc2InternetGateway_GetAll(t *testing.T) {

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

	igw := InternetGateway{
		Client: mockedInternetGateway{
			DescribeInternetGatewaysOutput: ec2.DescribeInternetGatewaysOutput{
				InternetGateways: []*ec2.InternetGateway{
					{
						InternetGatewayId: awsgo.String(gateway1),
						Tags: []*ec2.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName1),
							}, {
								Key:   awsgo.String(util.FirstSeenTagKey),
								Value: awsgo.String(util.FormatTimestamp(now.Add(1))),
							},
						},
					},
					{
						InternetGatewayId: awsgo.String(gateway2),
						Tags: []*ec2.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName2),
							}, {
								Key:   awsgo.String(util.FirstSeenTagKey),
								Value: awsgo.String(util.FormatTimestamp(now.Add(1))),
							},
						},
					},
				},
			},
		},
	}
	igw.BaseAwsResource.Init(nil)

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
					TimeAfter: awsgo.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := igw.getAll(tc.ctx, config.Config{
				InternetGateway: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}

}

func TestEc2InternetGateway_NukeAll(t *testing.T) {

	t.Parallel()

	var (
		gateway1 = "igw-0b44cfa6103932e1d001"
		gateway2 = "igw-0b44cfa6103932e1d002"
	)

	igw := InternetGateway{
		BaseAwsResource: BaseAwsResource{
			Nukables: map[string]error{
				gateway1: nil,
			},
		},
		Client: mockedInternetGateway{
			DescribeInternetGatewaysOutput: ec2.DescribeInternetGatewaysOutput{
				InternetGateways: []*ec2.InternetGateway{
					{
						InternetGatewayId: awsgo.String(gateway1),
						Attachments: []*ec2.InternetGatewayAttachment{
							{
								State: awsgo.String("testing-state"),
								VpcId: awsgo.String("test-gateway-vpc"),
							},
						},
					},
					{
						InternetGatewayId: awsgo.String(gateway2),
						Attachments: []*ec2.InternetGatewayAttachment{
							{
								State: awsgo.String("testing-state"),
								VpcId: awsgo.String("test-gateway-vpc"),
							},
						},
					},
				},
			},
			DeleteInternetGatewayOutput: ec2.DeleteInternetGatewayOutput{},
		},
	}

	err := igw.nukeAll([]*string{
		awsgo.String(gateway1),
		awsgo.String(gateway2),
	})
	require.NoError(t, err)
}
