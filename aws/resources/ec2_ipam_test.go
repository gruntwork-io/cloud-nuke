package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedIPAM struct {
	ec2iface.EC2API
	DescribeIpamsOutput          ec2.DescribeIpamsOutput
	DeleteIpamOutput             ec2.DeleteIpamOutput
	DescribeIpamScopesOutput     ec2.DescribeIpamScopesOutput
	DescribeIpamPoolsOutput      ec2.DescribeIpamPoolsOutput
	GetIpamPoolCidrsOutput       ec2.GetIpamPoolCidrsOutput
	DeprovisionIpamPoolCidrOuput ec2.DeprovisionIpamPoolCidrOutput
	GetIpamPoolAllocationsOutput ec2.GetIpamPoolAllocationsOutput
}

func (m mockedIPAM) DeleteIpamPoolWithContext(_ awsgo.Context, _ *ec2.DeleteIpamPoolInput, _ ...request.Option) (*ec2.DeleteIpamPoolOutput, error) {
	return nil, nil
}
func (m mockedIPAM) ReleaseIpamPoolAllocationWithContext(_ awsgo.Context, _ *ec2.ReleaseIpamPoolAllocationInput, _ ...request.Option) (*ec2.ReleaseIpamPoolAllocationOutput, error) {
	return nil, nil
}
func (m mockedIPAM) GetIpamPoolAllocationsWithContext(_ awsgo.Context, _ *ec2.GetIpamPoolAllocationsInput, _ ...request.Option) (*ec2.GetIpamPoolAllocationsOutput, error) {
	return &m.GetIpamPoolAllocationsOutput, nil
}
func (m mockedIPAM) DeprovisionIpamPoolCidrWithContext(_ awsgo.Context, _ *ec2.DeprovisionIpamPoolCidrInput, _ ...request.Option) (*ec2.DeprovisionIpamPoolCidrOutput, error) {
	return &m.DeprovisionIpamPoolCidrOuput, nil
}
func (m mockedIPAM) GetIpamPoolCidrsWithContext(_ awsgo.Context, _ *ec2.GetIpamPoolCidrsInput, _ ...request.Option) (*ec2.GetIpamPoolCidrsOutput, error) {
	return &m.GetIpamPoolCidrsOutput, nil
}
func (m mockedIPAM) DescribeIpamScopesWithContext(_ awsgo.Context, _ *ec2.DescribeIpamScopesInput, _ ...request.Option) (*ec2.DescribeIpamScopesOutput, error) {
	return &m.DescribeIpamScopesOutput, nil
}
func (m mockedIPAM) DescribeIpamPoolsWithContext(_ awsgo.Context, _ *ec2.DescribeIpamPoolsInput, _ ...request.Option) (*ec2.DescribeIpamPoolsOutput, error) {
	return &m.DescribeIpamPoolsOutput, nil
}
func (m mockedIPAM) DescribeIpamsWithContext(_ awsgo.Context, _ *ec2.DescribeIpamsInput, _ ...request.Option) (*ec2.DescribeIpamsOutput, error) {
	return &m.DescribeIpamsOutput, nil
}
func (m mockedIPAM) DescribeIpamsPagesWithContext(_ awsgo.Context, _ *ec2.DescribeIpamsInput, callback func(*ec2.DescribeIpamsOutput, bool) bool, _ ...request.Option) error {
	callback(&m.DescribeIpamsOutput, true)
	return nil
}
func (m mockedIPAM) DeleteIpamWithContext(_ awsgo.Context, _ *ec2.DeleteIpamInput, _ ...request.Option) (*ec2.DeleteIpamOutput, error) {
	return &m.DeleteIpamOutput, nil
}

func TestIPAM_GetAll(t *testing.T) {
	t.Parallel()

	// Set excludeFirstSeenTag to false for testing
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	var (
		now       = time.Now()
		testId1   = "ipam-0dfc56f901b2c3462"
		testId2   = "ipam-0dfc56f901b2c3463"
		testName1 = "test-ipam-id1"
		testName2 = "test-ipam-id2"
	)

	ipam := EC2IPAMs{
		Client: mockedIPAM{
			DescribeIpamsOutput: ec2.DescribeIpamsOutput{
				Ipams: []*ec2.Ipam{
					{
						IpamId: &testId1,
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
						IpamId: &testId2,
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
				EC2IPAM: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(ids))
		})
	}
}

func TestIPAM_NukeAll(t *testing.T) {
	t.Parallel()

	var (
		now       = time.Now()
		testId1   = "ipam-0dfc56f901b2c3462"
		testId2   = "ipam-0dfc56f901b2c3463"
		testName1 = "test-ipam-id1"
		testName2 = "test-ipam-id2"

		publicScope = "ipam-scope-001277b300c015f14"
		// privateScope = "ipam-scope-0d49ce2576b99615a"

		publicScopeARN = "arn:aws:ec2::499213733106:ipam-scope/ipam-scope-001277b300c015f14"
	)

	ipam := EC2IPAMs{
		Client: mockedIPAM{
			DeleteIpamOutput: ec2.DeleteIpamOutput{},
			DescribeIpamsOutput: ec2.DescribeIpamsOutput{
				Ipams: []*ec2.Ipam{
					{
						IpamId:               &testId1,
						PublicDefaultScopeId: &publicScope,
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
						IpamId:               &testId2,
						PublicDefaultScopeId: &publicScope,
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
			DescribeIpamScopesOutput: ec2.DescribeIpamScopesOutput{
				IpamScopes: []*ec2.IpamScope{
					{
						IpamScopeId:  aws.String(publicScope),
						IpamScopeArn: aws.String(publicScopeARN),
					},
				},
			},
			DescribeIpamPoolsOutput: ec2.DescribeIpamPoolsOutput{
				IpamPools: []*ec2.IpamPool{
					{
						IpamPoolId: aws.String(testId1),
					},
				},
			},
			GetIpamPoolCidrsOutput: ec2.GetIpamPoolCidrsOutput{
				IpamPoolCidrs: []*ec2.IpamPoolCidr{
					{
						Cidr: aws.String("10.0.0.0/24"),
					},
				},
			},
			GetIpamPoolAllocationsOutput: ec2.GetIpamPoolAllocationsOutput{
				IpamPoolAllocations: []*ec2.IpamPoolAllocation{
					{
						ResourceType:         aws.String("custom"),
						IpamPoolAllocationId: aws.String("ipam-pool-alloc-001277b300c015f14"),
					},
				},
			},
		},
	}

	err := ipam.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
