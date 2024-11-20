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

type mockedNetworkFirewallTLSConfig struct {
	NetworkFirewallTLSConfigAPI
	ListTLSInspectionConfigurationsOutput    networkfirewall.ListTLSInspectionConfigurationsOutput
	DescribeTLSInspectionConfigurationOutput map[string]networkfirewall.DescribeTLSInspectionConfigurationOutput
	DeleteTLSInspectionConfigurationOutput   networkfirewall.DeleteTLSInspectionConfigurationOutput
}

func (m mockedNetworkFirewallTLSConfig) DescribeTLSInspectionConfiguration(ctx context.Context, params *networkfirewall.DescribeTLSInspectionConfigurationInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeTLSInspectionConfigurationOutput, error) {
	raw := aws.ToString(params.TLSInspectionConfigurationArn)
	v, ok := m.DescribeTLSInspectionConfigurationOutput[raw]
	if !ok {
		return nil, fmt.Errorf("unable to describe the %s", raw)
	}
	return &v, nil
}

func (m mockedNetworkFirewallTLSConfig) DeleteTLSInspectionConfiguration(ctx context.Context, params *networkfirewall.DeleteTLSInspectionConfigurationInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteTLSInspectionConfigurationOutput, error) {
	return &m.DeleteTLSInspectionConfigurationOutput, nil
}

func (m mockedNetworkFirewallTLSConfig) ListTLSInspectionConfigurations(ctx context.Context, params *networkfirewall.ListTLSInspectionConfigurationsInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListTLSInspectionConfigurationsOutput, error) {
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
				TLSInspectionConfigurations: []types.TLSInspectionConfigurationMetadata{
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
			DescribeTLSInspectionConfigurationOutput: map[string]networkfirewall.DescribeTLSInspectionConfigurationOutput{
				testId1: {
					TLSInspectionConfigurationResponse: &types.TLSInspectionConfigurationResponse{
						TLSInspectionConfigurationName: aws.String(testName1),
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
					TLSInspectionConfigurationResponse: &types.TLSInspectionConfigurationResponse{
						TLSInspectionConfigurationName: aws.String(testName2),
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
				NetworkFirewallTLSConfig: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
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
