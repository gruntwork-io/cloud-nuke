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

type mockedNetworkFirewallPolicy struct {
	NetworkFirewallPolicyAPI
	ListOutput     networkfirewall.ListFirewallPoliciesOutput
	DescribeOutput map[string]networkfirewall.DescribeFirewallPolicyOutput
	DeleteOutput   networkfirewall.DeleteFirewallPolicyOutput
}

func (m mockedNetworkFirewallPolicy) ListFirewallPolicies(ctx context.Context, params *networkfirewall.ListFirewallPoliciesInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListFirewallPoliciesOutput, error) {
	return &m.ListOutput, nil
}

func (m mockedNetworkFirewallPolicy) DescribeFirewallPolicy(ctx context.Context, params *networkfirewall.DescribeFirewallPolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeFirewallPolicyOutput, error) {
	arn := aws.ToString(params.FirewallPolicyArn)
	if output, ok := m.DescribeOutput[arn]; ok {
		return &output, nil
	}
	return &networkfirewall.DescribeFirewallPolicyOutput{}, nil
}

func (m mockedNetworkFirewallPolicy) DeleteFirewallPolicy(ctx context.Context, params *networkfirewall.DeleteFirewallPolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteFirewallPolicyOutput, error) {
	return &m.DeleteOutput, nil
}

func (m mockedNetworkFirewallPolicy) TagResource(ctx context.Context, params *networkfirewall.TagResourceInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.TagResourceOutput, error) {
	return &networkfirewall.TagResourceOutput{}, nil
}

func TestNetworkFirewallPolicy_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testName1 := "test-policy-1"
	testName2 := "test-policy-2"
	testArn1 := "arn:aws:network-firewall:us-east-1:123456789012:stateless-rulegroup/policy-1"
	testArn2 := "arn:aws:network-firewall:us-east-1:123456789012:stateless-rulegroup/policy-2"

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	mock := mockedNetworkFirewallPolicy{
		ListOutput: networkfirewall.ListFirewallPoliciesOutput{
			FirewallPolicies: []types.FirewallPolicyMetadata{
				{Arn: aws.String(testArn1), Name: aws.String(testName1)},
				{Arn: aws.String(testArn2), Name: aws.String(testName2)},
			},
		},
		DescribeOutput: map[string]networkfirewall.DescribeFirewallPolicyOutput{
			testArn1: {
				FirewallPolicyResponse: &types.FirewallPolicyResponse{
					FirewallPolicyName:   aws.String(testName1),
					NumberOfAssociations: aws.Int32(0),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName1)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
			},
			testArn2: {
				FirewallPolicyResponse: &types.FirewallPolicyResponse{
					FirewallPolicyName:   aws.String(testName2),
					NumberOfAssociations: aws.Int32(0),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName2)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now.Add(1 * time.Hour)))},
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
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				},
			},
			expected: []string{testName1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listNetworkFirewallPolicies(ctx, mock, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestNetworkFirewallPolicy_SkipsAssociatedPolicies(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
	now := time.Now()

	testName := "associated-policy"
	testArn := "arn:aws:network-firewall:us-east-1:123456789012:stateless-rulegroup/associated"

	mock := mockedNetworkFirewallPolicy{
		ListOutput: networkfirewall.ListFirewallPoliciesOutput{
			FirewallPolicies: []types.FirewallPolicyMetadata{
				{Arn: aws.String(testArn), Name: aws.String(testName)},
			},
		},
		DescribeOutput: map[string]networkfirewall.DescribeFirewallPolicyOutput{
			testArn: {
				FirewallPolicyResponse: &types.FirewallPolicyResponse{
					FirewallPolicyName:   aws.String(testName),
					NumberOfAssociations: aws.Int32(1), // Associated with a firewall
					Tags: []types.Tag{
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
			},
		},
	}

	names, err := listNetworkFirewallPolicies(ctx, mock, resource.Scope{Region: "us-east-1"}, config.ResourceType{})
	require.NoError(t, err)
	require.Empty(t, names, "Policies with associations should be skipped")
}

func TestNetworkFirewallPolicy_NukeAll(t *testing.T) {
	t.Parallel()

	mock := mockedNetworkFirewallPolicy{
		DeleteOutput: networkfirewall.DeleteFirewallPolicyOutput{},
	}

	err := deleteNetworkFirewallPolicy(context.Background(), mock, aws.String("test-policy"))
	require.NoError(t, err)
}
