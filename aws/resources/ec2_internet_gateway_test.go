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

func (m mockedInternetGateway) DescribeInternetGateways(_ *ec2.DescribeInternetGatewaysInput) (*ec2.DescribeInternetGatewaysOutput, error) {
	return &m.DescribeInternetGatewaysOutput, nil
}

func (m mockedInternetGateway) DetachInternetGateway(_ *ec2.DetachInternetGatewayInput) (*ec2.DetachInternetGatewayOutput, error) {
	return &m.DetachInternetGatewayOutput, nil
}

func (m mockedInternetGateway) DeleteInternetGateway(_ *ec2.DeleteInternetGatewayInput) (*ec2.DeleteInternetGatewayOutput, error) {
	return &m.DeleteInternetGatewayOutput, nil
}

func TestEc2InternetGateway_GetAll(t *testing.T) {

	t.Parallel()

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
						InternetGatewayId: aws.String(gateway1),
						Tags: []*ec2.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName1),
							}, {
								Key:   aws.String(util.FirstSeenTagKey),
								Value: aws.String(util.FormatTimestamp(now.Add(1))),
							},
						},
					},
					{
						InternetGatewayId: aws.String(gateway2),
						Tags: []*ec2.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName2),
							}, {
								Key:   aws.String(util.FirstSeenTagKey),
								Value: aws.String(util.FormatTimestamp(now.Add(1))),
							},
						},
					},
				},
			},
		},
	}
	igw.BaseAwsResource.Init(nil)

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
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := igw.getAll(context.Background(), config.Config{
				InternetGateway: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
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
						InternetGatewayId: aws.String(gateway1),
						Attachments: []*ec2.InternetGatewayAttachment{
							{
								State: aws.String("testing-state"),
								VpcId: aws.String("test-gateway-vpc"),
							},
						},
					},
					{
						InternetGatewayId: aws.String(gateway2),
						Attachments: []*ec2.InternetGatewayAttachment{
							{
								State: aws.String("testing-state"),
								VpcId: aws.String("test-gateway-vpc"),
							},
						},
					},
				},
			},
			DeleteInternetGatewayOutput: ec2.DeleteInternetGatewayOutput{},
		},
	}

	err := igw.nukeAll([]*string{
		aws.String(gateway1),
		aws.String(gateway2),
	})
	require.NoError(t, err)
}
