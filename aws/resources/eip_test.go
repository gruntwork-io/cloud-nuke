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

type mockedEIPAddresses struct {
	EIPAddressesAPI
	ReleaseAddressOutput    ec2.ReleaseAddressOutput
	DescribeAddressesOutput ec2.DescribeAddressesOutput
}

func (m mockedEIPAddresses) ReleaseAddress(ctx context.Context, params *ec2.ReleaseAddressInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseAddressOutput, error) {
	return &m.ReleaseAddressOutput, nil
}

func (m mockedEIPAddresses) DescribeAddresses(ctx context.Context, params *ec2.DescribeAddressesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error) {
	return &m.DescribeAddressesOutput, nil
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
				Addresses: []types.Address{
					{
						AllocationId: aws.String(testAllocId1),
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
						AllocationId: aws.String(testAllocId2),
						Tags: []types.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String(testName2),
							},
							{
								Key:   aws.String(util.FirstSeenTagKey),
								Value: aws.String(util.FormatTimestamp(now.Add(1))),
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
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
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
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
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

	err := ea.nukeAll([]*string{aws.String("alloc1")})
	require.NoError(t, err)
}
