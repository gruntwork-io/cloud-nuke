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

type mockEC2IPAMScopeClient struct {
	DescribeIpamScopesOutput ec2.DescribeIpamScopesOutput
	DeleteIpamScopeOutput    ec2.DeleteIpamScopeOutput
}

func (m *mockEC2IPAMScopeClient) DescribeIpamScopes(ctx context.Context, params *ec2.DescribeIpamScopesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamScopesOutput, error) {
	return &m.DescribeIpamScopesOutput, nil
}

func (m *mockEC2IPAMScopeClient) DeleteIpamScope(ctx context.Context, params *ec2.DeleteIpamScopeInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamScopeOutput, error) {
	return &m.DeleteIpamScopeOutput, nil
}

func TestListEC2IPAMScopes(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testId1 := "ipam-scope-0dfc56f901b2c3462"
	testId2 := "ipam-scope-0dfc56f901b2c3463"
	testName1 := "test-ipam-id1"
	testName2 := "test-ipam-id2"

	mock := &mockEC2IPAMScopeClient{
		DescribeIpamScopesOutput: ec2.DescribeIpamScopesOutput{
			IpamScopes: []types.IpamScope{
				{
					IpamScopeId: aws.String(testId1),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(testName1)},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					IpamScopeId: aws.String(testId2),
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
			ids, err := listEC2IPAMScopes(ctx, mock, resource.Scope{}, tc.cfg)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestDeleteEC2IPAMScope(t *testing.T) {
	t.Parallel()

	mock := &mockEC2IPAMScopeClient{}
	err := deleteEC2IPAMScope(context.Background(), mock, aws.String("ipam-scope-test"))
	require.NoError(t, err)
}
