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

type mockEC2IPAMPoolClient struct {
	DescribeIpamPoolsOutput ec2.DescribeIpamPoolsOutput
	DeleteIpamPoolOutput    ec2.DeleteIpamPoolOutput
}

func (m *mockEC2IPAMPoolClient) DescribeIpamPools(ctx context.Context, params *ec2.DescribeIpamPoolsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamPoolsOutput, error) {
	return &m.DescribeIpamPoolsOutput, nil
}

func (m *mockEC2IPAMPoolClient) DeleteIpamPool(ctx context.Context, params *ec2.DeleteIpamPoolInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamPoolOutput, error) {
	return &m.DeleteIpamPoolOutput, nil
}

func TestListEC2IPAMPools(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testId1 := "ipam-pool-0dfc56f901b2c3462"
	testId2 := "ipam-pool-0dfc56f901b2c3463"
	testName1 := "test-ipam-pool-id1"
	testName2 := "test-ipam-pool-id2"

	mock := &mockEC2IPAMPoolClient{
		DescribeIpamPoolsOutput: ec2.DescribeIpamPoolsOutput{
			IpamPools: []types.IpamPool{
				{
					IpamPoolId: aws.String(testId1),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName1)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					IpamPoolId: aws.String(testId2),
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
			ids, err := listEC2IPAMPools(ctx, mock, resource.Scope{}, tc.cfg)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestDeleteEC2IPAMPool(t *testing.T) {
	t.Parallel()

	mock := &mockEC2IPAMPoolClient{}
	err := deleteEC2IPAMPool(context.Background(), mock, aws.String("ipam-pool-test"))
	require.NoError(t, err)
}
