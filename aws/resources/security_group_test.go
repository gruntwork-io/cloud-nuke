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

type mockedSecurityGroup struct {
	DescribeSecurityGroupsOutput        *ec2.DescribeSecurityGroupsOutput
	DeleteSecurityGroupOutput           *ec2.DeleteSecurityGroupOutput
	DescribeInstancesOutput             *ec2.DescribeInstancesOutput
	AuthorizeSecurityGroupEgressOutput  *ec2.AuthorizeSecurityGroupEgressOutput
	AuthorizeSecurityGroupIngressOutput *ec2.AuthorizeSecurityGroupIngressOutput
	CreateSecurityGroupOutput           *ec2.CreateSecurityGroupOutput
	DescribeAddressesOutput             *ec2.DescribeAddressesOutput
	ReleaseAddressOutput                *ec2.ReleaseAddressOutput
	RevokeSecurityGroupEgressOutput     *ec2.RevokeSecurityGroupEgressOutput
	RevokeSecurityGroupIngressOutput    *ec2.RevokeSecurityGroupIngressOutput
	TerminateInstancesOutput            *ec2.TerminateInstancesOutput
}

func (m *mockedSecurityGroup) DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	return m.DescribeSecurityGroupsOutput, nil
}

func (m *mockedSecurityGroup) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return m.DescribeInstancesOutput, nil
}

func (m *mockedSecurityGroup) DeleteSecurityGroup(ctx context.Context, params *ec2.DeleteSecurityGroupInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSecurityGroupOutput, error) {
	return m.DeleteSecurityGroupOutput, nil
}

func (m *mockedSecurityGroup) AuthorizeSecurityGroupEgress(ctx context.Context, params *ec2.AuthorizeSecurityGroupEgressInput, optFns ...func(*ec2.Options)) (*ec2.AuthorizeSecurityGroupEgressOutput, error) {
	return m.AuthorizeSecurityGroupEgressOutput, nil
}

func (m *mockedSecurityGroup) AuthorizeSecurityGroupIngress(ctx context.Context, params *ec2.AuthorizeSecurityGroupIngressInput, optFns ...func(*ec2.Options)) (*ec2.AuthorizeSecurityGroupIngressOutput, error) {
	return m.AuthorizeSecurityGroupIngressOutput, nil
}

func (m *mockedSecurityGroup) CreateSecurityGroup(ctx context.Context, params *ec2.CreateSecurityGroupInput, optFns ...func(*ec2.Options)) (*ec2.CreateSecurityGroupOutput, error) {
	return m.CreateSecurityGroupOutput, nil
}
func (m *mockedSecurityGroup) DescribeAddresses(ctx context.Context, params *ec2.DescribeAddressesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error) {
	return m.DescribeAddressesOutput, nil
}
func (m *mockedSecurityGroup) ReleaseAddress(ctx context.Context, params *ec2.ReleaseAddressInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseAddressOutput, error) {
	return m.ReleaseAddressOutput, nil
}
func (m *mockedSecurityGroup) RevokeSecurityGroupEgress(ctx context.Context, params *ec2.RevokeSecurityGroupEgressInput, optFns ...func(*ec2.Options)) (*ec2.RevokeSecurityGroupEgressOutput, error) {
	return m.RevokeSecurityGroupEgressOutput, nil
}
func (m *mockedSecurityGroup) RevokeSecurityGroupIngress(ctx context.Context, params *ec2.RevokeSecurityGroupIngressInput, optFns ...func(*ec2.Options)) (*ec2.RevokeSecurityGroupIngressOutput, error) {
	return m.RevokeSecurityGroupIngressOutput, nil
}
func (m *mockedSecurityGroup) TerminateInstances(ctx context.Context, params *ec2.TerminateInstancesInput, optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error) {
	return m.TerminateInstancesOutput, nil
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
		Client: &mockedSecurityGroup{
			DescribeSecurityGroupsOutput: &ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []types.SecurityGroup{
					{
						GroupId:   aws.String(testId1),
						GroupName: aws.String(testName1),
						Tags: []types.Tag{
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
						Tags: []types.Tag{
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
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func Test_NukeAll(t *testing.T) {

	t.Parallel()

	er := SecurityGroup{
		Client: &mockedSecurityGroup{
			DeleteSecurityGroupOutput: &ec2.DeleteSecurityGroupOutput{},
		},
	}

	identifiers := []*string{aws.String("sg-1")}
	err := er.nukeAll(identifiers)
	require.NoError(t, err)
}
