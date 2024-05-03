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

type mockedNetworkFirewallRuleGroup struct {
	networkfirewalliface.NetworkFirewallAPI
	DescribeRuleGroupOutput map[string]networkfirewall.DescribeRuleGroupOutput
	TagResourceOutput       networkfirewall.TagResourceOutput
	DeleteRuleGroupOutput   networkfirewall.DeleteRuleGroupOutput
	ListRuleGroupsOutput    networkfirewall.ListRuleGroupsOutput
}

func (m mockedNetworkFirewallRuleGroup) TagResource(*networkfirewall.TagResourceInput) (*networkfirewall.TagResourceOutput, error) {
	return &m.TagResourceOutput, nil
}

func (m mockedNetworkFirewallRuleGroup) DescribeRuleGroup(req *networkfirewall.DescribeRuleGroupInput) (*networkfirewall.DescribeRuleGroupOutput, error) {
	raw := awsgo.StringValue(req.RuleGroupArn)
	v, ok := m.DescribeRuleGroupOutput[raw]
	if !ok {
		return nil, fmt.Errorf("unable to describe the %s", raw)
	}
	return &v, nil
}

func (m mockedNetworkFirewallRuleGroup) DeleteRuleGroup(*networkfirewall.DeleteRuleGroupInput) (*networkfirewall.DeleteRuleGroupOutput, error) {
	return &m.DeleteRuleGroupOutput, nil
}

func (m mockedNetworkFirewallRuleGroup) ListRuleGroups(*networkfirewall.ListRuleGroupsInput) (*networkfirewall.ListRuleGroupsOutput, error) {
	return &m.ListRuleGroupsOutput, nil
}

func TestNetworkFirewallRuleGroup_GetAll(t *testing.T) {

	t.Parallel()

	var (
		now       = time.Now()
		testId1   = "test-network-rule-group-id1"
		testId2   = "test-network-firewall-id2"
		testName1 = "test-network-firewall-1"
		testName2 = "test-network-firewall-2"
		ctx       = context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
	)

	nfw := NetworkFirewallRuleGroup{
		RuleGroups: make(map[string]RuleGroup),
		Client: mockedNetworkFirewallRuleGroup{
			ListRuleGroupsOutput: networkfirewall.ListRuleGroupsOutput{
				RuleGroups: []*networkfirewall.RuleGroupMetadata{
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
			DescribeRuleGroupOutput: map[string]networkfirewall.DescribeRuleGroupOutput{
				testId1: {
					RuleGroupResponse: &networkfirewall.RuleGroupResponse{
						RuleGroupName: awsgo.String(testName1),
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
					RuleGroupResponse: &networkfirewall.RuleGroupResponse{
						RuleGroupName: awsgo.String(testName2),
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
				NetworkFirewallRuleGroup: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestNetworkFirewallRuleGroup_NukeAll(t *testing.T) {

	t.Parallel()

	ngw := NetworkFirewallRuleGroup{
		Client: mockedNetworkFirewallRuleGroup{
			DeleteRuleGroupOutput: networkfirewall.DeleteRuleGroupOutput{},
		},
		RuleGroups: map[string]RuleGroup{
			"test-001": {
				Name: awsgo.String("test-001"),
				Type: awsgo.String("stateless"),
			},
			"test-002": {
				Name: awsgo.String("test-002"),
				Type: awsgo.String("stateless"),
			},
		},
	}

	err := ngw.nukeAll([]*string{
		aws.String("test-001"),
		aws.String("test-002"),
	})
	require.NoError(t, err)
}
