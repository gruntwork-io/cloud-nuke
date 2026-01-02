package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockedNetworkFirewallResourcePolicy struct {
	NetworkFirewallResourcePolicyAPI
	ListFirewallPoliciesOutput   networkfirewall.ListFirewallPoliciesOutput
	ListRuleGroupsOutput         networkfirewall.ListRuleGroupsOutput
	DescribeResourcePolicyOutput networkfirewall.DescribeResourcePolicyOutput
	DeleteResourcePolicyOutput   networkfirewall.DeleteResourcePolicyOutput
}

func (m mockedNetworkFirewallResourcePolicy) ListFirewallPolicies(ctx context.Context, params *networkfirewall.ListFirewallPoliciesInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListFirewallPoliciesOutput, error) {
	return &m.ListFirewallPoliciesOutput, nil
}

func (m mockedNetworkFirewallResourcePolicy) ListRuleGroups(ctx context.Context, params *networkfirewall.ListRuleGroupsInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListRuleGroupsOutput, error) {
	return &m.ListRuleGroupsOutput, nil
}

func (m mockedNetworkFirewallResourcePolicy) DeleteResourcePolicy(ctx context.Context, params *networkfirewall.DeleteResourcePolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteResourcePolicyOutput, error) {
	return &m.DeleteResourcePolicyOutput, nil
}

func (m mockedNetworkFirewallResourcePolicy) DescribeResourcePolicy(ctx context.Context, params *networkfirewall.DescribeResourcePolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeResourcePolicyOutput, error) {
	return &m.DescribeResourcePolicyOutput, nil
}

func TestNetworkFirewallResourcePolicy_GetAll(t *testing.T) {
	t.Parallel()

	policy1 := "arn:aws:network-firewall:us-east-1:123456789012:firewall-policy/test-policy-1"
	policy2 := "arn:aws:network-firewall:us-east-1:123456789012:firewall-policy/test-policy-2"
	group1 := "arn:aws:network-firewall:us-east-1:123456789012:stateless-rulegroup/test-group-1"
	group2 := "arn:aws:network-firewall:us-east-1:123456789012:stateless-rulegroup/test-group-2"

	mock := mockedNetworkFirewallResourcePolicy{
		ListFirewallPoliciesOutput: networkfirewall.ListFirewallPoliciesOutput{
			FirewallPolicies: []types.FirewallPolicyMetadata{
				{Arn: aws.String(policy1)},
				{Arn: aws.String(policy2)},
			},
		},
		ListRuleGroupsOutput: networkfirewall.ListRuleGroupsOutput{
			RuleGroups: []types.RuleGroupMetadata{
				{Arn: aws.String(group1)},
				{Arn: aws.String(group2)},
			},
		},
		DescribeResourcePolicyOutput: networkfirewall.DescribeResourcePolicyOutput{
			Policy: aws.String(`{"Version":"2012-10-17","Statement":[]}`),
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{policy1, policy2, group1, group2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile("test-policy-1"),
					}},
				},
			},
			expected: []string{policy2, group1, group2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := listNetworkFirewallResourcePolicies(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestNetworkFirewallResourcePolicy_NukeAll(t *testing.T) {
	t.Parallel()

	mock := mockedNetworkFirewallResourcePolicy{
		DeleteResourcePolicyOutput: networkfirewall.DeleteResourcePolicyOutput{},
	}

	err := deleteNetworkFirewallResourcePolicy(context.Background(), mock, aws.String("arn:aws:network-firewall:us-east-1:123456789012:firewall-policy/test"))
	require.NoError(t, err)
}
