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

type mockedIPAM struct {
	EC2IPAMAPIaa
	DescribeIpamsOutput             ec2.DescribeIpamsOutput
	DeleteIpamOutput                ec2.DeleteIpamOutput
	GetIpamPoolCidrsOutput          ec2.GetIpamPoolCidrsOutput
	DeprovisionIpamPoolCidrOutput   ec2.DeprovisionIpamPoolCidrOutput
	GetIpamPoolAllocationsOutput    ec2.GetIpamPoolAllocationsOutput
	ReleaseIpamPoolAllocationOutput ec2.ReleaseIpamPoolAllocationOutput
	DescribeIpamScopesOutput        ec2.DescribeIpamScopesOutput
	DescribeIpamPoolsOutput         ec2.DescribeIpamPoolsOutput
	DeleteIpamPoolOutput            ec2.DeleteIpamPoolOutput
}

func (m mockedIPAM) DescribeIpams(ctx context.Context, params *ec2.DescribeIpamsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamsOutput, error) {
	return &m.DescribeIpamsOutput, nil
}

func (m mockedIPAM) DeleteIpam(ctx context.Context, params *ec2.DeleteIpamInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamOutput, error) {
	return &m.DeleteIpamOutput, nil
}

func (m mockedIPAM) GetIpamPoolCidrs(ctx context.Context, params *ec2.GetIpamPoolCidrsInput, optFns ...func(*ec2.Options)) (*ec2.GetIpamPoolCidrsOutput, error) {
	return &m.GetIpamPoolCidrsOutput, nil
}

func (m mockedIPAM) DeprovisionIpamPoolCidr(ctx context.Context, params *ec2.DeprovisionIpamPoolCidrInput, optFns ...func(*ec2.Options)) (*ec2.DeprovisionIpamPoolCidrOutput, error) {
	return &m.DeprovisionIpamPoolCidrOutput, nil
}

func (m mockedIPAM) GetIpamPoolAllocations(ctx context.Context, params *ec2.GetIpamPoolAllocationsInput, optFns ...func(*ec2.Options)) (*ec2.GetIpamPoolAllocationsOutput, error) {
	return &m.GetIpamPoolAllocationsOutput, nil
}

func (m mockedIPAM) ReleaseIpamPoolAllocation(ctx context.Context, params *ec2.ReleaseIpamPoolAllocationInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseIpamPoolAllocationOutput, error) {
	return &m.ReleaseIpamPoolAllocationOutput, nil
}

func (m mockedIPAM) DescribeIpamScopes(ctx context.Context, params *ec2.DescribeIpamScopesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamScopesOutput, error) {
	return &m.DescribeIpamScopesOutput, nil
}

func (m mockedIPAM) DescribeIpamPools(ctx context.Context, params *ec2.DescribeIpamPoolsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamPoolsOutput, error) {
	return &m.DescribeIpamPoolsOutput, nil
}

func (m mockedIPAM) DeleteIpamPool(ctx context.Context, params *ec2.DeleteIpamPoolInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamPoolOutput, error) {
	return &m.DeleteIpamPoolOutput, nil
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
				Ipams: []types.Ipam{
					{
						IpamId: &testId1,
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
						IpamId: &testId2,
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
				EC2IPAM: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
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
				Ipams: []types.Ipam{
					{
						IpamId:               &testId1,
						PublicDefaultScopeId: &publicScope,
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
						IpamId:               &testId2,
						PublicDefaultScopeId: &publicScope,
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
			DescribeIpamScopesOutput: ec2.DescribeIpamScopesOutput{
				IpamScopes: []types.IpamScope{
					{
						IpamScopeId:  aws.String(publicScope),
						IpamScopeArn: aws.String(publicScopeARN),
					},
				},
			},
			DescribeIpamPoolsOutput: ec2.DescribeIpamPoolsOutput{
				IpamPools: []types.IpamPool{
					{
						IpamPoolId: aws.String(testId1),
					},
				},
			},
			GetIpamPoolCidrsOutput: ec2.GetIpamPoolCidrsOutput{
				IpamPoolCidrs: []types.IpamPoolCidr{
					{
						Cidr: aws.String("10.0.0.0/24"),
					},
				},
			},
			GetIpamPoolAllocationsOutput: ec2.GetIpamPoolAllocationsOutput{
				IpamPoolAllocations: []types.IpamPoolAllocation{
					{
						ResourceType:         types.IpamPoolAllocationResourceTypeCustom,
						IpamPoolAllocationId: aws.String("ipam-pool-alloc-001277b300c015f14"),
					},
				},
			},
		},
	}

	err := ipam.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
