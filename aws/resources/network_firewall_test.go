package resources

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/networkfirewall"
	"github.com/aws/aws-sdk-go/service/networkfirewall/networkfirewalliface"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedNetworkFirewall struct {
	networkfirewalliface.NetworkFirewallAPI
	DeleteFirewallOutput   networkfirewall.DeleteFirewallOutput
	ListFirewallsOutput    networkfirewall.ListFirewallsOutput
	TagResourceOutput      networkfirewall.TagResourceOutput
	DescribeFirewallOutput map[string]networkfirewall.DescribeFirewallOutput
}

func (m mockedNetworkFirewall) TagResource(*networkfirewall.TagResourceInput) (*networkfirewall.TagResourceOutput, error) {
	return &m.TagResourceOutput, nil
}

func (m mockedNetworkFirewall) DeleteFirewallWithContext(_ awsgo.Context, _ *networkfirewall.DeleteFirewallInput, _ ...request.Option) (*networkfirewall.DeleteFirewallOutput, error) {
	return &m.DeleteFirewallOutput, nil
}

func (m mockedNetworkFirewall) ListFirewalls(*networkfirewall.ListFirewallsInput) (*networkfirewall.ListFirewallsOutput, error) {
	return &m.ListFirewallsOutput, nil
}

func (m mockedNetworkFirewall) DescribeFirewallWithContext(_ awsgo.Context, req *networkfirewall.DescribeFirewallInput, _ ...request.Option) (*networkfirewall.DescribeFirewallOutput, error) {
	raw := awsgo.StringValue(req.FirewallArn)
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
				Firewalls: []*networkfirewall.FirewallMetadata{
					{
						FirewallArn:  awsgo.String(testId1),
						FirewallName: awsgo.String(testName1),
					},
					{
						FirewallArn:  awsgo.String(testId2),
						FirewallName: awsgo.String(testName2),
					},
				},
			},
			DescribeFirewallOutput: map[string]networkfirewall.DescribeFirewallOutput{
				testId1: {
					Firewall: &networkfirewall.Firewall{
						FirewallName: awsgo.String(testName1),
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
					Firewall: &networkfirewall.Firewall{
						FirewallName: awsgo.String(testName2),
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
				NetworkFirewall: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
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
