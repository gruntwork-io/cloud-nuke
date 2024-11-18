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

type mockedIPAMScope struct {
	EC2IpamScopesAPI
	DescribeIpamScopesOutput ec2.DescribeIpamScopesOutput
	DeleteIpamScopeOutput    ec2.DeleteIpamScopeOutput
}

func (m mockedIPAMScope) DescribeIpamScopes(ctx context.Context, params *ec2.DescribeIpamScopesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamScopesOutput, error) {
	return &m.DescribeIpamScopesOutput, nil
}

func (m mockedIPAMScope) DeleteIpamScope(ctx context.Context, params *ec2.DeleteIpamScopeInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamScopeOutput, error) {
	return &m.DeleteIpamScopeOutput, nil
}

func TestIPAMScope_GetAll(t *testing.T) {
	t.Parallel()

	// Set excludeFirstSeenTag to false for testing
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	var (
		now       = time.Now()
		testId1   = "ipam-scope-0dfc56f901b2c3462"
		testId2   = "ipam-scope-0dfc56f901b2c3463"
		testName1 = "test-ipam-id1"
		testName2 = "test-ipam-id2"
	)

	ipam := EC2IpamScopes{
		Client: mockedIPAMScope{
			DescribeIpamScopesOutput: ec2.DescribeIpamScopesOutput{
				IpamScopes: []types.IpamScope{
					{
						IpamScopeId: &testId1,
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
						IpamScopeId: &testId2,
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
				EC2IPAMScope: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestIPAMScope_NukeAll(t *testing.T) {
	t.Parallel()

	ipam := EC2IpamScopes{
		Client: mockedIPAMScope{
			DeleteIpamScopeOutput: ec2.DeleteIpamScopeOutput{},
		},
	}

	err := ipam.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
