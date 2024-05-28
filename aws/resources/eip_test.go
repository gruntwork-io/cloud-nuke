package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedEIPAddresses struct {
	ec2iface.EC2API
	DescribeAddressesOutput ec2.DescribeAddressesOutput
	ReleaseAddressOutput    ec2.ReleaseAddressOutput
}

func (m mockedEIPAddresses) DescribeAddressesWithContext(_ awsgo.Context, _ *ec2.DescribeAddressesInput, _ ...request.Option) (*ec2.DescribeAddressesOutput, error) {
	return &m.DescribeAddressesOutput, nil
}

func (m mockedEIPAddresses) ReleaseAddressWithContext(_ awsgo.Context, _ *ec2.ReleaseAddressInput, _ ...request.Option) (*ec2.ReleaseAddressOutput, error) {
	return &m.ReleaseAddressOutput, nil
}

func TestEIPAddress_GetAll(t *testing.T) {

	t.Parallel()

	// Set excludeFirstSeenTag to false for testing
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	now := time.Now()
	testName1 := "test-eip1"
	testAllocId1 := "alloc1"
	testName2 := "test-eip2"
	testAllocId2 := "alloc2"
	ea := EIPAddresses{
		Client: &mockedEIPAddresses{
			DescribeAddressesOutput: ec2.DescribeAddressesOutput{
				Addresses: []*ec2.Address{
					{
						AllocationId: awsgo.String(testAllocId1),
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
						AllocationId: awsgo.String(testAllocId2),
						Tags: []*ec2.Tag{
							{
								Key:   awsgo.String("Name"),
								Value: awsgo.String(testName2),
							},
							{
								Key:   awsgo.String(util.FirstSeenTagKey),
								Value: awsgo.String(util.FormatTimestamp(now.Add(1))),
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
			expected:  []string{testAllocId1, testAllocId2},
		},
		"nameExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testAllocId2},
		},
		"timeAfterExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: awsgo.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := ea.getAll(tc.ctx, config.Config{
				ElasticIP: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}

}

func TestEIPAddress_NukeAll(t *testing.T) {

	t.Parallel()

	ea := EIPAddresses{
		Client: &mockedEIPAddresses{
			ReleaseAddressOutput: ec2.ReleaseAddressOutput{},
		},
	}

	err := ea.nukeAll([]*string{awsgo.String("alloc1")})
	require.NoError(t, err)
}
