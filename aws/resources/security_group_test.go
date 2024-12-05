package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedSecurityGroup struct {
	BaseAwsResource
	ec2iface.EC2API
	DescribeSecurityGroupsOutput ec2.DescribeSecurityGroupsOutput
	DeleteSecurityGroupOutput    ec2.DeleteSecurityGroupOutput
	DescribeInstancesOutput      ec2.DescribeInstancesOutput
}

func (m mockedSecurityGroup) DescribeSecurityGroupsWithContext(_ awsgo.Context, _ *ec2.DescribeSecurityGroupsInput, _ ...request.Option) (*ec2.DescribeSecurityGroupsOutput, error) {
	return &m.DescribeSecurityGroupsOutput, nil
}

func (m mockedSecurityGroup) DescribeInstancesWithContext(_ awsgo.Context, _ *ec2.DescribeInstancesInput, _ ...request.Option) (*ec2.DescribeInstancesOutput, error) {
	return &m.DescribeInstancesOutput, nil
}

func (m mockedSecurityGroup) DeleteSecurityGroupWithContext(_ awsgo.Context, _ *ec2.DeleteSecurityGroupInput, _ ...request.Option) (*ec2.DeleteSecurityGroupOutput, error) {
	return &m.DeleteSecurityGroupOutput, nil
}

func TestSecurityGroup_GetAll(t *testing.T) {

	var (
		testId1   = "sg-08f2b91f81265ab7b001"
		testId2   = "sg-08f2b91f81265ab7b002"
		testName1 = "cloud-nuke-sg-001"
		testName2 = "cloud-nuke-sg-002"
		now       = time.Now()
	)

	// Set excludeFirstSeenTag to false for testing
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	sg := SecurityGroup{
		BaseAwsResource: BaseAwsResource{
			Nukables: map[string]error{
				testId1: nil,
				testId2: nil,
			},
		},
		Client: &mockedSecurityGroup{
			DescribeSecurityGroupsOutput: ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []*ec2.SecurityGroup{
					{
						GroupId:   aws.String(testId1),
						GroupName: aws.String(testName1),
						Tags: []*ec2.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName1),
							},
							{
								Key:   aws.String(util.FirstSeenTagKey),
								Value: aws.String(util.FormatTimestamp(now)),
							},
						},
					},
					{
						GroupId:   aws.String(testId2),
						GroupName: aws.String(testName2),
						Tags: []*ec2.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName2),
							},
							{
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
		configObj config.EC2ResourceType
		expected  []string
	}{
		"emptyFilter": {
			ctx:       ctx,
			configObj: config.EC2ResourceType{},
			expected:  []string{testId1, testId2},
		},
		"nameExclusionFilter": {
			ctx: ctx,
			configObj: config.EC2ResourceType{
				ResourceType: config.ResourceType{
					ExcludeRule: config.FilterRule{
						NamesRegExp: []config.Expression{{
							RE: *regexp.MustCompile(testName1),
						}}},
				},
			},
			expected: []string{testId2},
		},
		"timeAfterExclusionFilter": {
			ctx: ctx,
			configObj: config.EC2ResourceType{
				ResourceType: config.ResourceType{
					ExcludeRule: config.FilterRule{
						TimeAfter: aws.Time(now),
					}},
			},
			expected: []string{testId1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			names, err := sg.getAll(tc.ctx, config.Config{
				SecurityGroup: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func Test_NukeAll(t *testing.T) {

	t.Parallel()

	er := SecurityGroup{
		Client: &mockedSecurityGroup{
			DeleteSecurityGroupOutput: ec2.DeleteSecurityGroupOutput{},
		},
	}

	identifiers := []*string{aws.String("sg-1")}
	err := er.nukeAll(identifiers)
	require.NoError(t, err)
}
