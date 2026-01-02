package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockEC2IPAMResourceDiscoveryClient struct {
	DescribeIpamResourceDiscoveriesOutput ec2.DescribeIpamResourceDiscoveriesOutput
	DeleteIpamResourceDiscoveryOutput     ec2.DeleteIpamResourceDiscoveryOutput
}

func (m *mockEC2IPAMResourceDiscoveryClient) DescribeIpamResourceDiscoveries(ctx context.Context, params *ec2.DescribeIpamResourceDiscoveriesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamResourceDiscoveriesOutput, error) {
	return &m.DescribeIpamResourceDiscoveriesOutput, nil
}

func (m *mockEC2IPAMResourceDiscoveryClient) DeleteIpamResourceDiscovery(ctx context.Context, params *ec2.DeleteIpamResourceDiscoveryInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamResourceDiscoveryOutput, error) {
	return &m.DeleteIpamResourceDiscoveryOutput, nil
}

func TestListEC2IPAMResourceDiscoveries(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testId1 := "ipam-res-disco-0dfc56f901b2c3462"
	testId2 := "ipam-res-disco-0dfc56f901b2c3463"
	testName1 := "test-ipam-resource-id1"
	testName2 := "test-ipam-resource-id2"

	mock := &mockEC2IPAMResourceDiscoveryClient{
		DescribeIpamResourceDiscoveriesOutput: ec2.DescribeIpamResourceDiscoveriesOutput{
			IpamResourceDiscoveries: []types.IpamResourceDiscovery{
				{
					IpamResourceDiscoveryId: aws.String(testId1),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName1)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					IpamResourceDiscoveryId: aws.String(testId2),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName2)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
			},
		},
	}

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	tests := map[string]struct {
		cfg      config.ResourceType
		expected []string
	}{
		"emptyFilter": {
			cfg:      config.ResourceType{},
			expected: []string{testId1, testId2},
		},
		"nameExclusionFilter": {
			cfg: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile(testName1)}},
				},
			},
			expected: []string{testId2},
		},
		"timeAfterExclusionFilter": {
			cfg: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				},
			},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := listEC2IPAMResourceDiscoveries(ctx, mock, resource.Scope{}, tc.cfg)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestDeleteEC2IPAMResourceDiscovery(t *testing.T) {
	t.Parallel()

	mock := &mockEC2IPAMResourceDiscoveryClient{}
	err := deleteEC2IPAMResourceDiscovery(context.Background(), mock, aws.String("ipam-res-disco-test"))
	require.NoError(t, err)
}
