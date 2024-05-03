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

type mockedNetworkFirewallTLSConfig struct {
	networkfirewalliface.NetworkFirewallAPI
	DescribeTLSInspectionConfigurationOutput map[string]networkfirewall.DescribeTLSInspectionConfigurationOutput
	TagResourceOutput                        networkfirewall.TagResourceOutput
	DeleteTLSInspectionConfigurationOutput   networkfirewall.DeleteTLSInspectionConfigurationOutput
	ListTLSInspectionConfigurationsOutput    networkfirewall.ListTLSInspectionConfigurationsOutput
}

func (m mockedNetworkFirewallTLSConfig) TagResource(*networkfirewall.TagResourceInput) (*networkfirewall.TagResourceOutput, error) {
	return &m.TagResourceOutput, nil
}

func (m mockedNetworkFirewallTLSConfig) DescribeTLSInspectionConfiguration(req *networkfirewall.DescribeTLSInspectionConfigurationInput) (*networkfirewall.DescribeTLSInspectionConfigurationOutput, error) {
	raw := awsgo.StringValue(req.TLSInspectionConfigurationArn)
	v, ok := m.DescribeTLSInspectionConfigurationOutput[raw]
	if !ok {
		return nil, fmt.Errorf("unable to describe the %s", raw)
	}
	return &v, nil
}

func (m mockedNetworkFirewallTLSConfig) DeleteTLSInspectionConfiguration(*networkfirewall.DeleteTLSInspectionConfigurationInput) (*networkfirewall.DeleteTLSInspectionConfigurationOutput, error) {
	return &m.DeleteTLSInspectionConfigurationOutput, nil
}

func (m mockedNetworkFirewallTLSConfig) ListTLSInspectionConfigurations(*networkfirewall.ListTLSInspectionConfigurationsInput) (*networkfirewall.ListTLSInspectionConfigurationsOutput, error) {
	return &m.ListTLSInspectionConfigurationsOutput, nil
}

func TestNetworkFirewallTLSConfig_GetAll(t *testing.T) {

	t.Parallel()

	var (
		now       = time.Now()
		testId1   = "test-network-firewall-id1"
		testId2   = "test-network-firewall-id2"
		testName1 = "test-network-firewall-1"
		testName2 = "test-network-firewall-2"
		ctx       = context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
	)

	nfw := NetworkFirewallTLSConfig{
		Client: mockedNetworkFirewallTLSConfig{
			ListTLSInspectionConfigurationsOutput: networkfirewall.ListTLSInspectionConfigurationsOutput{
				TLSInspectionConfigurations: []*networkfirewall.TLSInspectionConfigurationMetadata{
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
			DescribeTLSInspectionConfigurationOutput: map[string]networkfirewall.DescribeTLSInspectionConfigurationOutput{
				testId1: {
					TLSInspectionConfigurationResponse: &networkfirewall.TLSInspectionConfigurationResponse{
						TLSInspectionConfigurationName: awsgo.String(testName1),
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
					TLSInspectionConfigurationResponse: &networkfirewall.TLSInspectionConfigurationResponse{
						TLSInspectionConfigurationName: awsgo.String(testName2),
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
				NetworkFirewallTLSConfig: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestNetworkFirewallTLSConfig_NukeAll(t *testing.T) {

	t.Parallel()

	ngw := NetworkFirewallTLSConfig{
		Client: mockedNetworkFirewallTLSConfig{
			DeleteTLSInspectionConfigurationOutput: networkfirewall.DeleteTLSInspectionConfigurationOutput{},
		},
	}

	err := ngw.nukeAll([]*string{
		aws.String("test-001"),
		aws.String("test-002"),
	})
	require.NoError(t, err)
}
