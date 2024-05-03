package resources

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/networkfirewall"
	"github.com/aws/aws-sdk-go/service/networkfirewall/networkfirewalliface"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedNetworkFirewallPolicy struct {
	networkfirewalliface.NetworkFirewallAPI
	DeleteFirewallPolicyOutput   networkfirewall.DeleteFirewallPolicyOutput
	ListFirewallPoliciesOutput   networkfirewall.ListFirewallPoliciesOutput
	TagResourceOutput            networkfirewall.TagResourceOutput
	DescribeFirewallPolicyOutput map[string]networkfirewall.DescribeFirewallPolicyOutput
}

func (m mockedNetworkFirewallPolicy) TagResource(*networkfirewall.TagResourceInput) (*networkfirewall.TagResourceOutput, error) {
	return &m.TagResourceOutput, nil
}

func (m mockedNetworkFirewallPolicy) DeleteFirewallPolicy(*networkfirewall.DeleteFirewallPolicyInput) (*networkfirewall.DeleteFirewallPolicyOutput, error) {
	return &m.DeleteFirewallPolicyOutput, nil
}

func (m mockedNetworkFirewallPolicy) ListFirewallPolicies(*networkfirewall.ListFirewallPoliciesInput) (*networkfirewall.ListFirewallPoliciesOutput, error) {
	return &m.ListFirewallPoliciesOutput, nil
}

func (m mockedNetworkFirewallPolicy) DescribeFirewallPolicy(req *networkfirewall.DescribeFirewallPolicyInput) (*networkfirewall.DescribeFirewallPolicyOutput, error) {
	raw := awsgo.StringValue(req.FirewallPolicyArn)
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
				FirewallPolicies: []*networkfirewall.FirewallPolicyMetadata{
					{
						Arn:  awsgo.String(testId1),
						Name: awsgo.String(testName1),
					},
					{
						Arn:  awsgo.String(testId2),
						Name: awsgo.String(testName2),
					},
				},
			},
			DescribeFirewallPolicyOutput: map[string]networkfirewall.DescribeFirewallPolicyOutput{
				testId1: {
					FirewallPolicyResponse: &networkfirewall.FirewallPolicyResponse{
						FirewallPolicyName: awsgo.String(testName1),
						Tags: []*networkfirewall.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName1),
							}, {
								Key:   awsgo.String(util.FirstSeenTagKey),
								Value: awsgo.String(util.FormatTimestamp(now)),
							},
						},
					},
				},
				testId2: {
					FirewallPolicyResponse: &networkfirewall.FirewallPolicyResponse{
						FirewallPolicyName: awsgo.String(testName2),
						Tags: []*networkfirewall.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName2),
							}, {
								Key:   awsgo.String(util.FirstSeenTagKey),
								Value: awsgo.String(util.FormatTimestamp(now.Add(1 * time.Hour))),
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
					TimeAfter: awsgo.Time(now),
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
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
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
