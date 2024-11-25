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

type mockedNetworkFirewallRuleGroup struct {
	NetworkFirewallRuleGroupAPI
	ListRuleGroupsOutput    networkfirewall.ListRuleGroupsOutput
	DescribeRuleGroupOutput map[string]networkfirewall.DescribeRuleGroupOutput
	DeleteRuleGroupOutput   networkfirewall.DeleteRuleGroupOutput
}

func (m mockedNetworkFirewallRuleGroup) DescribeRuleGroup(ctx context.Context, params *networkfirewall.DescribeRuleGroupInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeRuleGroupOutput, error) {
	raw := aws.ToString(params.RuleGroupArn)
	v, ok := m.DescribeRuleGroupOutput[raw]
	if !ok {
		return nil, fmt.Errorf("unable to describe the %s", raw)
	}
	return &v, nil
}

func (m mockedNetworkFirewallRuleGroup) DeleteRuleGroup(ctx context.Context, params *networkfirewall.DeleteRuleGroupInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteRuleGroupOutput, error) {
	return &m.DeleteRuleGroupOutput, nil
}

func (m mockedNetworkFirewallRuleGroup) ListRuleGroups(ctx context.Context, params *networkfirewall.ListRuleGroupsInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListRuleGroupsOutput, error) {
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
				RuleGroups: []types.RuleGroupMetadata{
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
			DescribeRuleGroupOutput: map[string]networkfirewall.DescribeRuleGroupOutput{
				testId1: {
					RuleGroupResponse: &types.RuleGroupResponse{
						RuleGroupName: aws.String(testName1),
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
					RuleGroupResponse: &types.RuleGroupResponse{
						RuleGroupName: aws.String(testName2),
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
				NetworkFirewallRuleGroup: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
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
				Name: aws.String("test-001"),
				Type: aws.String("stateless"),
			},
			"test-002": {
				Name: aws.String("test-002"),
				Type: aws.String("stateless"),
			},
		},
	}

	err := ngw.nukeAll([]*string{
		aws.String("test-001"),
		aws.String("test-002"),
	})
	require.NoError(t, err)
}
