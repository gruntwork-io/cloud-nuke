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

type mockedIPAMPools struct {
	ec2iface.EC2API
	DescribeIpamPoolsOutput ec2.DescribeIpamPoolsOutput
	DeleteIpamPoolOutput    ec2.DeleteIpamPoolOutput
}

func (m mockedIPAMPools) DescribeIpamPoolsPages(input *ec2.DescribeIpamPoolsInput, callback func(*ec2.DescribeIpamPoolsOutput, bool) bool) error {
	callback(&m.DescribeIpamPoolsOutput, true)
	return nil
}
func (m mockedIPAMPools) DeleteIpamPool(params *ec2.DeleteIpamPoolInput) (*ec2.DeleteIpamPoolOutput, error) {
	return &m.DeleteIpamPoolOutput, nil
}

func TestIPAMPool_GetAll(t *testing.T) {
	t.Parallel()

	var (
		now       = time.Now()
		testId1   = "ipam-pool-0dfc56f901b2c3462"
		testId2   = "ipam-pool-0dfc56f901b2c3463"
		testName1 = "test-ipam-pool-id1"
		testName2 = "test-ipam-pool-id2"
	)

	ipam := EC2IPAMPool{
		Client: mockedIPAMPools{
			DescribeIpamPoolsOutput: ec2.DescribeIpamPoolsOutput{
				IpamPools: []*ec2.IpamPool{
					{
						IpamPoolId: aws.String(testId1),
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
						IpamPoolId: aws.String(testId2),
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
				EC2IPAMPool: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(ids))
		})
	}
}

func TestIPAMPool_NukeAll(t *testing.T) {
	t.Parallel()

	ipam := EC2IPAMPool{
		Client: mockedIPAMPools{
			DeleteIpamPoolOutput: ec2.DeleteIpamPoolOutput{},
		},
	}

	err := ipam.nukeAll([]*string{aws.String("test")})
	require.NoError(t, err)
}
