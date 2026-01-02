package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockNetworkFirewallRuleGroupClient struct {
	NetworkFirewallRuleGroupAPI
	ListRuleGroupsOutput    networkfirewall.ListRuleGroupsOutput
	DescribeRuleGroupOutput map[string]networkfirewall.DescribeRuleGroupOutput
	DeleteRuleGroupOutput   networkfirewall.DeleteRuleGroupOutput
}

func (m *mockNetworkFirewallRuleGroupClient) ListRuleGroups(_ context.Context, _ *networkfirewall.ListRuleGroupsInput, _ ...func(*networkfirewall.Options)) (*networkfirewall.ListRuleGroupsOutput, error) {
	return &m.ListRuleGroupsOutput, nil
}

func (m *mockNetworkFirewallRuleGroupClient) DescribeRuleGroup(_ context.Context, params *networkfirewall.DescribeRuleGroupInput, _ ...func(*networkfirewall.Options)) (*networkfirewall.DescribeRuleGroupOutput, error) {
	arn := aws.ToString(params.RuleGroupArn)
	if output, ok := m.DescribeRuleGroupOutput[arn]; ok {
		return &output, nil
	}
	return &networkfirewall.DescribeRuleGroupOutput{}, nil
}

func (m *mockNetworkFirewallRuleGroupClient) DeleteRuleGroup(_ context.Context, _ *networkfirewall.DeleteRuleGroupInput, _ ...func(*networkfirewall.Options)) (*networkfirewall.DeleteRuleGroupOutput, error) {
	return &m.DeleteRuleGroupOutput, nil
}

func TestNetworkFirewallRuleGroup_List(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testArn1 := "arn:aws:network-firewall:us-east-1:123456789012:stateless-rulegroup/test-group-1"
	testArn2 := "arn:aws:network-firewall:us-east-1:123456789012:stateful-rulegroup/test-group-2"
	testName1 := "test-group-1"
	testName2 := "test-group-2"

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	mockClient := &mockNetworkFirewallRuleGroupClient{
		ListRuleGroupsOutput: networkfirewall.ListRuleGroupsOutput{
			RuleGroups: []types.RuleGroupMetadata{
				{Arn: aws.String(testArn1), Name: aws.String(testName1)},
				{Arn: aws.String(testArn2), Name: aws.String(testName2)},
			},
		},
		DescribeRuleGroupOutput: map[string]networkfirewall.DescribeRuleGroupOutput{
			testArn1: {
				RuleGroupResponse: &types.RuleGroupResponse{
					RuleGroupName:        aws.String(testName1),
					Type:                 types.RuleGroupTypeStateless,
					NumberOfAssociations: aws.Int32(0),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName1)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
			},
			testArn2: {
				RuleGroupResponse: &types.RuleGroupResponse{
					RuleGroupName:        aws.String(testName2),
					Type:                 types.RuleGroupTypeStateful,
					NumberOfAssociations: aws.Int32(0),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName2)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now.Add(time.Hour)))},
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
			expected:  []string{testArn1, testArn2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testArn2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				},
			},
			expected: []string{testArn1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			arns, err := listNetworkFirewallRuleGroups(ctx, mockClient, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(arns))
		})
	}
}

func TestNetworkFirewallRuleGroup_SkipsInUse(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, true)
	testArn := "arn:aws:network-firewall:us-east-1:123456789012:stateless-rulegroup/in-use-group"
	testName := "in-use-group"

	mockClient := &mockNetworkFirewallRuleGroupClient{
		ListRuleGroupsOutput: networkfirewall.ListRuleGroupsOutput{
			RuleGroups: []types.RuleGroupMetadata{
				{Arn: aws.String(testArn), Name: aws.String(testName)},
			},
		},
		DescribeRuleGroupOutput: map[string]networkfirewall.DescribeRuleGroupOutput{
			testArn: {
				RuleGroupResponse: &types.RuleGroupResponse{
					RuleGroupName:        aws.String(testName),
					Type:                 types.RuleGroupTypeStateless,
					NumberOfAssociations: aws.Int32(1), // In use
				},
			},
		},
	}

	arns, err := listNetworkFirewallRuleGroups(ctx, mockClient, resource.Scope{Region: "us-east-1"}, config.ResourceType{})
	require.NoError(t, err)
	require.Empty(t, arns)
}

func TestNetworkFirewallRuleGroup_Delete(t *testing.T) {
	t.Parallel()

	testArn := "arn:aws:network-firewall:us-east-1:123456789012:stateless-rulegroup/test-delete-group"

	mockClient := &mockNetworkFirewallRuleGroupClient{
		DeleteRuleGroupOutput: networkfirewall.DeleteRuleGroupOutput{},
	}

	err := deleteNetworkFirewallRuleGroup(context.Background(), mockClient, aws.String(testArn))
	require.NoError(t, err)
}
