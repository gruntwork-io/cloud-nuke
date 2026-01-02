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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

// mockedSecurityGroup implements SecurityGroupAPI for testing.
type mockedSecurityGroup struct {
	SecurityGroupAPI
	DescribeSecurityGroupsOutput     ec2.DescribeSecurityGroupsOutput
	DeleteSecurityGroupOutput        ec2.DeleteSecurityGroupOutput
	DescribeInstancesOutput          ec2.DescribeInstancesOutput
	DescribeAddressesOutput          ec2.DescribeAddressesOutput
	ReleaseAddressOutput             ec2.ReleaseAddressOutput
	RevokeSecurityGroupEgressOutput  ec2.RevokeSecurityGroupEgressOutput
	RevokeSecurityGroupIngressOutput ec2.RevokeSecurityGroupIngressOutput
	TerminateInstancesOutput         ec2.TerminateInstancesOutput
}

func (m mockedSecurityGroup) DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	return &m.DescribeSecurityGroupsOutput, nil
}

func (m mockedSecurityGroup) DeleteSecurityGroup(ctx context.Context, params *ec2.DeleteSecurityGroupInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSecurityGroupOutput, error) {
	return &m.DeleteSecurityGroupOutput, nil
}

func (m mockedSecurityGroup) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return &m.DescribeInstancesOutput, nil
}

func (m mockedSecurityGroup) DescribeAddresses(ctx context.Context, params *ec2.DescribeAddressesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error) {
	return &m.DescribeAddressesOutput, nil
}

func (m mockedSecurityGroup) ReleaseAddress(ctx context.Context, params *ec2.ReleaseAddressInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseAddressOutput, error) {
	return &m.ReleaseAddressOutput, nil
}

func (m mockedSecurityGroup) RevokeSecurityGroupEgress(ctx context.Context, params *ec2.RevokeSecurityGroupEgressInput, optFns ...func(*ec2.Options)) (*ec2.RevokeSecurityGroupEgressOutput, error) {
	return &m.RevokeSecurityGroupEgressOutput, nil
}

func (m mockedSecurityGroup) RevokeSecurityGroupIngress(ctx context.Context, params *ec2.RevokeSecurityGroupIngressInput, optFns ...func(*ec2.Options)) (*ec2.RevokeSecurityGroupIngressOutput, error) {
	return &m.RevokeSecurityGroupIngressOutput, nil
}

func (m mockedSecurityGroup) TerminateInstances(ctx context.Context, params *ec2.TerminateInstancesInput, optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error) {
	return &m.TerminateInstancesOutput, nil
}

func TestSecurityGroup_List(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testID1 := "sg-08f2b91f81265ab7b001"
	testID2 := "sg-08f2b91f81265ab7b002"
	testName1 := "cloud-nuke-sg-001"
	testName2 := "cloud-nuke-sg-002"

	mockClient := mockedSecurityGroup{
		DescribeSecurityGroupsOutput: ec2.DescribeSecurityGroupsOutput{
			SecurityGroups: []types.SecurityGroup{
				{
					GroupId:   aws.String(testID1),
					GroupName: aws.String(testName1),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName1)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					GroupId:   aws.String(testID2),
					GroupName: aws.String(testName2),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName2)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now.Add(1 * time.Hour)))},
					},
				},
			},
		},
	}

	// Set excludeFirstSeenTag to false for testing
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	tests := map[string]struct {
		configObj   config.ResourceType
		defaultOnly bool
		expected    []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testID1, testID2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testID2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				},
			},
			expected: []string{testID1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := listSecurityGroups(ctx, mockClient, tc.configObj, tc.defaultOnly)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestSecurityGroup_List_SkipsDefault(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mockClient := mockedSecurityGroup{
		DescribeSecurityGroupsOutput: ec2.DescribeSecurityGroupsOutput{
			SecurityGroups: []types.SecurityGroup{
				{
					GroupId:   aws.String("sg-default"),
					GroupName: aws.String("default"),
					Tags: []types.Tag{
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					GroupId:   aws.String("sg-custom"),
					GroupName: aws.String("custom-sg"),
					Tags: []types.Tag{
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
			},
		},
	}

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	// Without defaultOnly, should skip "default" security group
	ids, err := listSecurityGroups(ctx, mockClient, config.ResourceType{}, false)
	require.NoError(t, err)
	require.Equal(t, []string{"sg-custom"}, aws.ToStringSlice(ids))
}

func TestSecurityGroup_Nuke(t *testing.T) {
	t.Parallel()

	mockClient := mockedSecurityGroup{
		DescribeInstancesOutput:      ec2.DescribeInstancesOutput{},
		DescribeSecurityGroupsOutput: ec2.DescribeSecurityGroupsOutput{},
		DeleteSecurityGroupOutput:    ec2.DeleteSecurityGroupOutput{},
	}

	err := nukeSecurityGroup(context.Background(), mockClient, aws.String("sg-test"), false)
	require.NoError(t, err)
}

func TestSecurityGroup_NukeAll(t *testing.T) {
	t.Parallel()

	mockClient := mockedSecurityGroup{
		DescribeInstancesOutput:      ec2.DescribeInstancesOutput{},
		DescribeSecurityGroupsOutput: ec2.DescribeSecurityGroupsOutput{},
		DeleteSecurityGroupOutput:    ec2.DeleteSecurityGroupOutput{},
	}

	results := nukeSecurityGroups(
		context.Background(),
		mockClient,
		resource.Scope{Region: "us-east-1"},
		"security-group",
		[]*string{aws.String("sg-1"), aws.String("sg-2")},
		false,
	)

	require.Len(t, results, 2)
	for _, r := range results {
		require.NoError(t, r.Error)
	}
}

func TestFindMatchingGroupRules(t *testing.T) {
	t.Parallel()

	targetID := aws.String("sg-target")

	tests := map[string]struct {
		permissions []types.IpPermission
		expectMatch bool
		expectCount int
	}{
		"noMatch": {
			permissions: []types.IpPermission{
				{
					IpProtocol: aws.String("-1"),
					UserIdGroupPairs: []types.UserIdGroupPair{
						{GroupId: aws.String("sg-other")},
					},
				},
			},
			expectMatch: false,
			expectCount: 0,
		},
		"singleMatch": {
			permissions: []types.IpPermission{
				{
					IpProtocol: aws.String("-1"),
					UserIdGroupPairs: []types.UserIdGroupPair{
						{GroupId: aws.String("sg-target")},
					},
				},
			},
			expectMatch: true,
			expectCount: 1,
		},
		"mixedRules": {
			permissions: []types.IpPermission{
				{
					IpProtocol: aws.String("tcp"),
					FromPort:   aws.Int32(22),
					ToPort:     aws.Int32(22),
					UserIdGroupPairs: []types.UserIdGroupPair{
						{GroupId: aws.String("sg-other")},
					},
				},
				{
					IpProtocol: aws.String("tcp"),
					FromPort:   aws.Int32(443),
					ToPort:     aws.Int32(443),
					UserIdGroupPairs: []types.UserIdGroupPair{
						{GroupId: aws.String("sg-target")},
						{GroupId: aws.String("sg-other2")},
					},
				},
			},
			expectMatch: true,
			expectCount: 1, // Only the permission with sg-target match
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			hasMatch, perms := findMatchingGroupRules(targetID, tc.permissions)
			require.Equal(t, tc.expectMatch, hasMatch)
			require.Len(t, perms, tc.expectCount)
		})
	}
}
