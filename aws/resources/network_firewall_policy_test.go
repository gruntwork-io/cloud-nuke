package resources

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedNetworkFirewallPolicy struct {
	NetworkFirewallPolicyAPI
	ListFirewallPoliciesOutput   networkfirewall.ListFirewallPoliciesOutput
	DescribeFirewallPolicyOutput map[string]networkfirewall.DescribeFirewallPolicyOutput
	DeleteFirewallPolicyOutput   networkfirewall.DeleteFirewallPolicyOutput
}

func (m mockedNetworkFirewallPolicy) DeleteFirewallPolicy(ctx context.Context, params *networkfirewall.DeleteFirewallPolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteFirewallPolicyOutput, error) {
	return &m.DeleteFirewallPolicyOutput, nil
}

func (m mockedNetworkFirewallPolicy) ListFirewallPolicies(ctx context.Context, params *networkfirewall.ListFirewallPoliciesInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListFirewallPoliciesOutput, error) {
	return &m.ListFirewallPoliciesOutput, nil
}

func (m mockedNetworkFirewallPolicy) DescribeFirewallPolicy(ctx context.Context, params *networkfirewall.DescribeFirewallPolicyInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeFirewallPolicyOutput, error) {
	raw := aws.ToString(params.FirewallPolicyArn)
	v, ok := m.DescribeFirewallPolicyOutput[raw]
	if !ok {
		return nil, fmt.Errorf("unable to describe the %s", raw)
	}
	return &v, nil
}

func TestNetworkFirewallPolicy_GetAll(t *testing.T) {

	t.Parallel()

	var (
		now       = time.Now()
		testId1   = "test-network-firewall-id1"
		testId2   = "test-network-firewall-id2"
		testName1 = "test-network-firewall-1"
		testName2 = "test-network-firewall-2"
		ctx       = context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
	)

	nfw := NetworkFirewallPolicy{
		Client: mockedNetworkFirewallPolicy{
			ListFirewallPoliciesOutput: networkfirewall.ListFirewallPoliciesOutput{
				FirewallPolicies: []types.FirewallPolicyMetadata{
					{
						Arn:  aws.String(testId1),
						Name: aws.String(testName1),
					},
					{
						Arn:  aws.String(testId2),
						Name: aws.String(testName2),
					},
				},
			},
			DescribeFirewallPolicyOutput: map[string]networkfirewall.DescribeFirewallPolicyOutput{
				testId1: {
					FirewallPolicyResponse: &types.FirewallPolicyResponse{
						FirewallPolicyName: aws.String(testName1),
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
				},
				testId2: {
					FirewallPolicyResponse: &types.FirewallPolicyResponse{
						FirewallPolicyName: aws.String(testName2),
						Tags: []types.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName2),
							}, {
								Key:   aws.String(util.FirstSeenTagKey),
								Value: aws.String(util.FormatTimestamp(now.Add(1 * time.Hour))),
							},
						},
					},
				},
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
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testName1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := nfw.getAll(ctx, config.Config{
				NetworkFirewallPolicy: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestNetworkFirewallPolicy_NukeAll(t *testing.T) {

	t.Parallel()

	ngw := NetworkFirewallPolicy{
		Client: mockedNetworkFirewallPolicy{
			DeleteFirewallPolicyOutput: networkfirewall.DeleteFirewallPolicyOutput{},
		},
	}

	err := ngw.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
