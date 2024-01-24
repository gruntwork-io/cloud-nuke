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

func (m mockedIPAM) DeleteIpamPool(input *ec2.DeleteIpamPoolInput) (*ec2.DeleteIpamPoolOutput, error) {
	return nil, nil
}
func (m mockedIPAM) ReleaseIpamPoolAllocation(input *ec2.ReleaseIpamPoolAllocationInput) (*ec2.ReleaseIpamPoolAllocationOutput, error) {
	return nil, nil
}
func (m mockedIPAM) GetIpamPoolAllocations(input *ec2.GetIpamPoolAllocationsInput) (*ec2.GetIpamPoolAllocationsOutput, error) {
	return &m.GetIpamPoolAllocationsOutput, nil
}
func (m mockedIPAM) DeprovisionIpamPoolCidr(input *ec2.DeprovisionIpamPoolCidrInput) (*ec2.DeprovisionIpamPoolCidrOutput, error) {
	return &m.DeprovisionIpamPoolCidrOuput, nil
}
func (m mockedIPAM) GetIpamPoolCidrs(input *ec2.GetIpamPoolCidrsInput) (*ec2.GetIpamPoolCidrsOutput, error) {
	return &m.GetIpamPoolCidrsOutput, nil
}
func (m mockedIPAM) DescribeIpamScopes(input *ec2.DescribeIpamScopesInput) (*ec2.DescribeIpamScopesOutput, error) {
	return &m.DescribeIpamScopesOutput, nil
}
func (m mockedIPAM) DescribeIpamPools(input *ec2.DescribeIpamPoolsInput) (*ec2.DescribeIpamPoolsOutput, error) {
	return &m.DescribeIpamPoolsOutput, nil
}
func (m mockedIPAM) DescribeIpams(input *ec2.DescribeIpamsInput) (*ec2.DescribeIpamsOutput, error) {
	return &m.DescribeIpamsOutput, nil
}
func (m mockedIPAM) DescribeIpamsPages(input *ec2.DescribeIpamsInput, callback func(*ec2.DescribeIpamsOutput, bool) bool) error {
	callback(&m.DescribeIpamsOutput, true)
	return nil
}
func (m mockedIPAM) DeleteIpam(params *ec2.DeleteIpamInput) (*ec2.DeleteIpamOutput, error) {
	return &m.DeleteIpamOutput, nil
}

func TestIPAM_GetAll(t *testing.T) {
	t.Parallel()

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
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testId1, testId2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testId2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := ipam.getAll(context.Background(), config.Config{
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
