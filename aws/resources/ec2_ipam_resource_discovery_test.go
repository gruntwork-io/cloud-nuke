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
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedIPAMResourceDiscovery struct {
	EC2IPAMResourceDiscoveryAPI
	DescribeIpamResourceDiscoveriesOutput ec2.DescribeIpamResourceDiscoveriesOutput
	DeleteIpamResourceDiscoveryOutput     ec2.DeleteIpamResourceDiscoveryOutput
}

func (m mockedIPAMResourceDiscovery) DescribeIpamResourceDiscoveries(ctx context.Context, params *ec2.DescribeIpamResourceDiscoveriesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamResourceDiscoveriesOutput, error) {
	return &m.DescribeIpamResourceDiscoveriesOutput, nil
}

func (m mockedIPAMResourceDiscovery) DeleteIpamResourceDiscovery(ctx context.Context, params *ec2.DeleteIpamResourceDiscoveryInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamResourceDiscoveryOutput, error) {
	return &m.DeleteIpamResourceDiscoveryOutput, nil
}

func TestIPAMRDiscovery_GetAll(t *testing.T) {
	t.Parallel()

	// Set excludeFirstSeenTag to false for testing
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	var (
		now       = time.Now()
		testId1   = "ipam-res-disco-0dfc56f901b2c3462"
		testId2   = "ipam-res-disco-0dfc56f901b2c3463"
		testName1 = "test-ipam-resource-id1"
		testName2 = "test-ipam-resource-id2"
	)

	ipam := EC2IPAMResourceDiscovery{
		Client: mockedIPAMResourceDiscovery{
			DescribeIpamResourceDiscoveriesOutput: ec2.DescribeIpamResourceDiscoveriesOutput{
				IpamResourceDiscoveries: []types.IpamResourceDiscovery{
					{
						IpamResourceDiscoveryId: aws.String(testId1),
						Tags: []types.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName1),
							},
							{
								Key:   aws.String(util.FirstSeenTagKey),
								Value: aws.String(util.FormatTimestamp(now)),
							},
						},
					},
					{
						IpamResourceDiscoveryId: aws.String(testId2),
						Tags: []types.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName2),
							},
							{
								Key:   aws.String(util.FirstSeenTagKey),
								Value: aws.String(util.FormatTimestamp(now)),
							},
						},
					},
				},
			},
		},
	}

	tests := map[string]struct {
		ctx       context.Context
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			ctx:       ctx,
			configObj: config.ResourceType{},
			expected:  []string{testId1, testId2},
		},
		"nameExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testId2},
		},
		"timeAfterExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := ipam.getAll(tc.ctx, config.Config{
				EC2IPAMResourceDiscovery: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestIPAMRDiscovery_NukeAll(t *testing.T) {
	t.Parallel()

	ipam := EC2IPAMResourceDiscovery{
		Client: mockedIPAMResourceDiscovery{
			DeleteIpamResourceDiscoveryOutput: ec2.DeleteIpamResourceDiscoveryOutput{},
		},
	}

	err := ipam.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
