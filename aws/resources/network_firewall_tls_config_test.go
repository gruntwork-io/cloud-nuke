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
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockNetworkFirewallTLSConfigClient struct {
	ListOutput     networkfirewall.ListTLSInspectionConfigurationsOutput
	DescribeOutput map[string]networkfirewall.DescribeTLSInspectionConfigurationOutput
	DeleteOutput   networkfirewall.DeleteTLSInspectionConfigurationOutput
}

func (m *mockNetworkFirewallTLSConfigClient) ListTLSInspectionConfigurations(ctx context.Context, params *networkfirewall.ListTLSInspectionConfigurationsInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListTLSInspectionConfigurationsOutput, error) {
	return &m.ListOutput, nil
}

func (m *mockNetworkFirewallTLSConfigClient) DescribeTLSInspectionConfiguration(ctx context.Context, params *networkfirewall.DescribeTLSInspectionConfigurationInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeTLSInspectionConfigurationOutput, error) {
	arn := aws.ToString(params.TLSInspectionConfigurationArn)
	if output, ok := m.DescribeOutput[arn]; ok {
		return &output, nil
	}
	return nil, fmt.Errorf("TLS config not found: %s", arn)
}

func (m *mockNetworkFirewallTLSConfigClient) DeleteTLSInspectionConfiguration(ctx context.Context, params *networkfirewall.DeleteTLSInspectionConfigurationInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteTLSInspectionConfigurationOutput, error) {
	return &m.DeleteOutput, nil
}

func TestListNetworkFirewallTLSConfigs(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testArn1 := "arn:aws:network-firewall:us-east-1:123456789:tls-config/config-1"
	testArn2 := "arn:aws:network-firewall:us-east-1:123456789:tls-config/config-2"
	testName1 := "test-tls-config-1"
	testName2 := "test-tls-config-2"
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	mock := &mockNetworkFirewallTLSConfigClient{
		ListOutput: networkfirewall.ListTLSInspectionConfigurationsOutput{
			TLSInspectionConfigurations: []types.TLSInspectionConfigurationMetadata{
				{Arn: aws.String(testArn1), Name: aws.String(testName1)},
				{Arn: aws.String(testArn2), Name: aws.String(testName2)},
			},
		},
		DescribeOutput: map[string]networkfirewall.DescribeTLSInspectionConfigurationOutput{
			testArn1: {
				TLSInspectionConfigurationResponse: &types.TLSInspectionConfigurationResponse{
					TLSInspectionConfigurationName: aws.String(testName1),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName1)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
			},
			testArn2: {
				TLSInspectionConfigurationResponse: &types.TLSInspectionConfigurationResponse{
					TLSInspectionConfigurationName: aws.String(testName2),
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
			names, err := listNetworkFirewallTLSConfigs(ctx, mock, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteNetworkFirewallTLSConfig(t *testing.T) {
	t.Parallel()

	mock := &mockNetworkFirewallTLSConfigClient{}
	err := deleteNetworkFirewallTLSConfig(context.Background(), mock, aws.String("test-config"))
	require.NoError(t, err)
}
