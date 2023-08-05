package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

type mockedEC2VPCs struct {
	ec2iface.EC2API
	DescribeVpcsOutput ec2.DescribeVpcsOutput
	DeleteVpcOutput    ec2.DeleteVpcOutput
}

func (m mockedEC2VPCs) DescribeVpcs(input *ec2.DescribeVpcsInput) (*ec2.DescribeVpcsOutput, error) {
	return &m.DescribeVpcsOutput, nil
}

func (m mockedEC2VPCs) DeleteVpc(input *ec2.DeleteVpcInput) (*ec2.DeleteVpcOutput, error) {
	return &m.DeleteVpcOutput, nil
}

func TestEC2VPC_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testName1 := "test-vpc-name1"
	testName2 := "test-vpc-name2"
	now := time.Now()
	testId1 := "test-vpc-id1"
	testId2 := "test-vpc-id2"
	vpc := EC2VPCs{
		Client: mockedEC2VPCs{
			DescribeVpcsOutput: ec2.DescribeVpcsOutput{
				Vpcs: []*ec2.Vpc{
					{
						VpcId: awsgo.String(testId1),
						Tags: []*ec2.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName1),
							},
							{
								Key:   awsgo.String(util.FirstSeenTagKey),
								Value: awsgo.String(util.FormatTimestampTag(now)),
							},
						},
					},
					{
						VpcId: awsgo.String(testId2),
						Tags: []*ec2.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName2),
							},
							{
								Key:   awsgo.String(util.FirstSeenTagKey),
								Value: awsgo.String(util.FormatTimestampTag(now.Add(1))),
							},
						},
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
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := vpc.getAll(config.Config{
				VPC: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}
}

func TestEC2VPC_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	vpc := EC2VPCs{
		Client: mockedEC2VPCs{
			DeleteVpcOutput: ec2.DeleteVpcOutput{},
		},
	}

	err := vpc.nukeAll([]string{"test-vpc-id1", "test-vpc-id2"})
	require.NoError(t, err)
}
