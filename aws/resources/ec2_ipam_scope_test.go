package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedIPAMScope struct {
	ec2iface.EC2API
	DescribeIpamScopesOutput ec2.DescribeIpamScopesOutput
	DeleteIpamScopeOutput    ec2.DeleteIpamScopeOutput
}

func (m mockedIPAMScope) DescribeIpamScopesPages(input *ec2.DescribeIpamScopesInput, callback func(*ec2.DescribeIpamScopesOutput, bool) bool) error {
	callback(&m.DescribeIpamScopesOutput, true)
	return nil
}
func (m mockedIPAMScope) DeleteIpamScope(params *ec2.DeleteIpamScopeInput) (*ec2.DeleteIpamScopeOutput, error) {
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
				IpamScopes: []*ec2.IpamScope{
					{
						IpamScopeId: &testId1,
						Tags: []*ec2.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName1),
							},
							{
								Key:   awsgo.String(util.FirstSeenTagKey),
								Value: awsgo.String(util.FormatTimestamp(now)),
							},
						},
					},
					{
						IpamScopeId: &testId2,
						Tags: []*ec2.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName2),
							},
							{
								Key:   awsgo.String(util.FirstSeenTagKey),
								Value: awsgo.String(util.FormatTimestamp(now)),
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
			require.Equal(t, tc.expected, awsgo.StringValueSlice(ids))
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
