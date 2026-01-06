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

type mockNetworkFirewallClient struct {
	ListFirewallsOutput    networkfirewall.ListFirewallsOutput
	DescribeFirewallOutput map[string]networkfirewall.DescribeFirewallOutput
	DeleteFirewallOutput   networkfirewall.DeleteFirewallOutput
}

func (m *mockNetworkFirewallClient) ListFirewalls(ctx context.Context, params *networkfirewall.ListFirewallsInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListFirewallsOutput, error) {
	return &m.ListFirewallsOutput, nil
}

func (m *mockNetworkFirewallClient) DescribeFirewall(ctx context.Context, params *networkfirewall.DescribeFirewallInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeFirewallOutput, error) {
	key := aws.ToString(params.FirewallArn)
	if key == "" {
		key = aws.ToString(params.FirewallName)
	}
	if output, ok := m.DescribeFirewallOutput[key]; ok {
		return &output, nil
	}
	return nil, fmt.Errorf("firewall not found: %s", key)
}

func (m *mockNetworkFirewallClient) DeleteFirewall(ctx context.Context, params *networkfirewall.DeleteFirewallInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DeleteFirewallOutput, error) {
	return &m.DeleteFirewallOutput, nil
}

func TestListNetworkFirewalls(t *testing.T) {
	t.Parallel()

	testName1 := "test-firewall-1"
	testName2 := "test-firewall-2"
	testArn1 := "arn:aws:network-firewall:us-east-1:123456789012:firewall/test-firewall-1"
	testArn2 := "arn:aws:network-firewall:us-east-1:123456789012:firewall/test-firewall-2"
	now := time.Now()

	mock := &mockNetworkFirewallClient{
		ListFirewallsOutput: networkfirewall.ListFirewallsOutput{
			Firewalls: []types.FirewallMetadata{
				{FirewallArn: aws.String(testArn1), FirewallName: aws.String(testName1)},
				{FirewallArn: aws.String(testArn2), FirewallName: aws.String(testName2)},
			},
		},
		DescribeFirewallOutput: map[string]networkfirewall.DescribeFirewallOutput{
			testArn1: {
				Firewall: &types.Firewall{
					FirewallName: aws.String(testName1),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName1)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
			},
			testArn2: {
				Firewall: &types.Firewall{
					FirewallName: aws.String(testName2),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName2)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now.Add(1 * time.Hour)))},
					},
				},
			},
		},
	}

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

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
					TimeAfter: aws.Time(now.Add(30 * time.Minute)),
				},
			},
			expected: []string{testName1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listNetworkFirewalls(ctx, mock, resource.Scope{Region: "us-east-1"}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteNetworkFirewall(t *testing.T) {
	t.Parallel()

	mock := &mockNetworkFirewallClient{}
	err := deleteNetworkFirewall(context.Background(), mock, aws.String("test-firewall"))
	require.NoError(t, err)
}

func TestVerifyNetworkFirewallPermissions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		deleteProtection bool
		expectErr        bool
	}{
		"not protected":    {deleteProtection: false, expectErr: false},
		"delete protected": {deleteProtection: true, expectErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mock := &mockNetworkFirewallClient{
				DescribeFirewallOutput: map[string]networkfirewall.DescribeFirewallOutput{
					"test-firewall": {
						Firewall: &types.Firewall{
							FirewallName:     aws.String("test-firewall"),
							DeleteProtection: tc.deleteProtection,
						},
					},
				},
			}

			err := verifyNetworkFirewallPermissions(context.Background(), mock, aws.String("test-firewall"))
			if tc.expectErr {
				require.ErrorIs(t, err, util.ErrDeleteProtectionEnabled)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
