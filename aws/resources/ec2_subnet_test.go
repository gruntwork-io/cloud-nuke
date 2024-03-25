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

type mockedEC2Subnets struct {
	ec2iface.EC2API
	DescribeSubnetsOutput ec2.DescribeSubnetsOutput
	DeleteSubnetOutput    ec2.DeleteSubnetOutput
}

func (m mockedEC2Subnets) DescribeSubnetsPages(_ *ec2.DescribeSubnetsInput, callback func(pages *ec2.DescribeSubnetsOutput, lastPage bool) bool) error {
	callback(&m.DescribeSubnetsOutput, true)
	return nil
}
func (m mockedEC2Subnets) DeleteSubnet(_ *ec2.DeleteSubnetInput) (*ec2.DeleteSubnetOutput, error) {
	return &m.DeleteSubnetOutput, nil
}

func TestEc2Subnets_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	var (
		now       = time.Now()
		subnet1   = "subnet-0631b58700ba3db41"
		testName1 = "cloud-nuke-subnet-001"
		subnet2   = "subnet-0631b58700ba3db42"
		testName2 = "cloud-nuke-subnet-002"
	)

	ec2subnet := EC2Subnet{
		Client: mockedEC2Subnets{
			DescribeSubnetsOutput: ec2.DescribeSubnetsOutput{
				Subnets: []*ec2.Subnet{
					{
						SubnetId: aws.String(subnet1),
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
						SubnetId: aws.String(subnet2),
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
	ec2subnet.BaseAwsResource.Init(nil)

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{subnet1, subnet2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{subnet2},
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
			names, err := ec2subnet.getAll(context.Background(), config.Config{
				EC2Subnet: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}

}

func TestEc2Subnet_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	tgw := EC2Subnet{
		Client: mockedEC2Subnets{
			DeleteSubnetOutput: ec2.DeleteSubnetOutput{},
		},
	}

	err := tgw.nukeAll([]*string{aws.String("test-gateway")})
	require.NoError(t, err)
}
