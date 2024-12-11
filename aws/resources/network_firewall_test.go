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

type mockedNetworkFirewall struct {
	NetworkFirewallAPI

	ListFirewallsOutput    networkfirewall.ListFirewallsOutput
	DescribeFirewallOutput map[string]networkfirewall.DescribeFirewallOutput
	DeleteFirewallOutput   networkfirewall.DeleteFirewallOutput
}

func (m mockedNetworkFirewall) DeleteFirewall(ctx context.Context, params *networkfirewall.DeleteFirewallInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteFirewallOutput, error) {
	return &m.DeleteFirewallOutput, nil
}

func (m mockedNetworkFirewall) ListFirewalls(ctx context.Context, params *networkfirewall.ListFirewallsInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListFirewallsOutput, error) {
	return &m.ListFirewallsOutput, nil
}

func (m mockedNetworkFirewall) DescribeFirewall(ctx context.Context, params *networkfirewall.DescribeFirewallInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeFirewallOutput, error) {
	raw := aws.ToString(params.FirewallArn)
	v, ok := m.DescribeFirewallOutput[raw]
	if !ok {
		return nil, fmt.Errorf("unable to describe the %s", raw)
	}
	return &v, nil
}

func TestNetworkFirewall_GetAll(t *testing.T) {

	t.Parallel()

	var (
		now       = time.Now()
		testId1   = "test-network-firewall-id1"
		testId2   = "test-network-firewall-id2"
		testName1 = "test-network-firewall-1"
		testName2 = "test-network-firewall-2"
		ctx       = context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
	)

	nfw := NetworkFirewall{
		Client: mockedNetworkFirewall{
			ListFirewallsOutput: networkfirewall.ListFirewallsOutput{
				Firewalls: []types.FirewallMetadata{
					{
						FirewallArn:  aws.String(testId1),
						FirewallName: aws.String(testName1),
					},
					{
						FirewallArn:  aws.String(testId2),
						FirewallName: aws.String(testName2),
					},
				},
			},
			DescribeFirewallOutput: map[string]networkfirewall.DescribeFirewallOutput{
				testId1: {
					Firewall: &types.Firewall{
						FirewallName: aws.String(testName1),
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
					Firewall: &types.Firewall{
						FirewallName: aws.String(testName2),
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
				NetworkFirewall: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestNetworkFirewall_NukeAll(t *testing.T) {

	t.Parallel()

	ngw := NetworkFirewall{
		Client: mockedNetworkFirewall{
			DeleteFirewallOutput: networkfirewall.DeleteFirewallOutput{},
		},
	}

	err := ngw.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
