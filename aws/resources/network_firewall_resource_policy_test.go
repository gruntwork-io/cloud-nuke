package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/networkfirewall"
	"github.com/aws/aws-sdk-go/service/networkfirewall/networkfirewalliface"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedNetworkFirewallResourcePolicy struct {
	networkfirewalliface.NetworkFirewallAPI
	ListFirewallPoliciesOutput   networkfirewall.ListFirewallPoliciesOutput
	ListRuleGroupsOutput         networkfirewall.ListRuleGroupsOutput
	DeleteResourcePolicyOutput   networkfirewall.DeleteResourcePolicyOutput
	DescribeResourcePolicyOutput networkfirewall.DescribeResourcePolicyOutput
}

func (m mockedNetworkFirewallResourcePolicy) ListFirewallPoliciesWithContext(_ awsgo.Context, _ *networkfirewall.ListFirewallPoliciesInput, _ ...request.Option) (*networkfirewall.ListFirewallPoliciesOutput, error) {
	return &m.ListFirewallPoliciesOutput, nil
}

func (m mockedNetworkFirewallResourcePolicy) ListRuleGroupsWithContext(awsgo.Context, *networkfirewall.ListRuleGroupsInput, ...request.Option) (*networkfirewall.ListRuleGroupsOutput, error) {
	return &m.ListRuleGroupsOutput, nil
}
func (m mockedNetworkFirewallResourcePolicy) DeleteResourcePolicyWithContext(awsgo.Context, *networkfirewall.DeleteResourcePolicyInput, ...request.Option) (*networkfirewall.DeleteResourcePolicyOutput, error) {
	return &m.DeleteResourcePolicyOutput, nil
}

func (m mockedNetworkFirewallResourcePolicy) DescribeResourcePolicyWithContext(awsgo.Context, *networkfirewall.DescribeResourcePolicyInput, ...request.Option) (*networkfirewall.DescribeResourcePolicyOutput, error) {
	return &m.DescribeResourcePolicyOutput, nil
}

func TestNetworkFirewallResourcePolicy_GetAll(t *testing.T) {

	t.Parallel()

	var (
		policy1 = "test-network-firewall-policy-1"
		policy2 = "test-network-firewall-policy-2"
		group1  = "test-network-firewall-group-1"
		group2  = "test-network-firewall-group-2"
	)

	nfw := NetworkFirewallResourcePolicy{
		Client: mockedNetworkFirewallResourcePolicy{
			ListFirewallPoliciesOutput: networkfirewall.ListFirewallPoliciesOutput{
				FirewallPolicies: []*networkfirewall.FirewallPolicyMetadata{
					{
						Arn: awsgo.String(policy1),
					},
					{
						Arn: awsgo.String(policy2),
					},
				},
			},
			ListRuleGroupsOutput: networkfirewall.ListRuleGroupsOutput{
				RuleGroups: []*networkfirewall.RuleGroupMetadata{
					{
						Arn: awsgo.String(group1),
					},
					{
						Arn: awsgo.String(group2),
					},
				},
			},
			DescribeResourcePolicyOutput: networkfirewall.DescribeResourcePolicyOutput{
				Policy: aws.String("policy-statements"),
			},
		},
	}

	nfw.BaseAwsResource.Init(nil)

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
						RE: *regexp.MustCompile(policy1),
					}}},
			},
			expected: []string{policy2, group1, group2},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := nfw.getAll(context.Background(), config.Config{
				NetworkFirewallResourcePolicy: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestNetworkFirewallResourcePolicy_NukeAll(t *testing.T) {

	t.Parallel()

	ngw := NetworkFirewallResourcePolicy{
		Client: mockedNetworkFirewallResourcePolicy{
			DeleteResourcePolicyOutput: networkfirewall.DeleteResourcePolicyOutput{},
		},
	}

	err := ngw.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
