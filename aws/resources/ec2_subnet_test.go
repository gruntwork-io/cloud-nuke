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

type mockedEC2Subnets struct {
	EC2SubnetAPI
	DescribeSubnetsOutput ec2.DescribeSubnetsOutput
	DeleteSubnetOutput    ec2.DeleteSubnetOutput
}

func (m mockedEC2Subnets) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	return &m.DescribeSubnetsOutput, nil
}
func (m mockedEC2Subnets) DeleteSubnet(ctx context.Context, params *ec2.DeleteSubnetInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSubnetOutput, error) {
	return &m.DeleteSubnetOutput, nil
}

func TestEc2Subnets_GetAll(t *testing.T) {

	t.Parallel()

	// Set excludeFirstSeenTag to false for testing
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

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
				Subnets: []types.Subnet{
					{
						SubnetId: aws.String(subnet1),
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
						SubnetId: aws.String(subnet2),
						Tags: []types.Tag{
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

	tests := map[string]struct {
		ctx       context.Context
		configObj config.EC2ResourceType
		expected  []string
	}{
		"emptyFilter": {
			ctx:       ctx,
			configObj: config.EC2ResourceType{},
			expected:  []string{subnet1, subnet2},
		},
		"nameExclusionFilter": {
			ctx: ctx,
			configObj: config.EC2ResourceType{
				ResourceType: config.ResourceType{
					ExcludeRule: config.FilterRule{
						NamesRegExp: []config.Expression{{
							RE: *regexp.MustCompile(testName1),
						}},
					},
				},
			},
			expected: []string{subnet2},
		},
		"timeAfterExclusionFilter": {
			ctx: ctx,
			configObj: config.EC2ResourceType{
				ResourceType: config.ResourceType{
					ExcludeRule: config.FilterRule{
						TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
					}},
			},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := ec2subnet.getAll(tc.ctx, config.Config{
				EC2Subnet: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}

}

func TestEc2Subnet_NukeAll(t *testing.T) {

	t.Parallel()

	tgw := EC2Subnet{
		Client: mockedEC2Subnets{
			DeleteSubnetOutput: ec2.DeleteSubnetOutput{},
		},
	}

	err := tgw.nukeAll([]*string{aws.String("test-gateway")})
	require.NoError(t, err)
}
