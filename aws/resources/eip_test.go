package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/andrewderr/cloud-nuke-a1/config"
	"github.com/andrewderr/cloud-nuke-a1/telemetry"
	"github.com/andrewderr/cloud-nuke-a1/util"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/stretchr/testify/require"
)

type mockedEIPAddresses struct {
	ec2iface.EC2API
	DescribeAddressesOutput ec2.DescribeAddressesOutput
	ReleaseAddressOutput    ec2.ReleaseAddressOutput
}

func (m mockedEIPAddresses) DescribeAddresses(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error) {
	return &m.DescribeAddressesOutput, nil
}

func (m mockedEIPAddresses) ReleaseAddress(*ec2.ReleaseAddressInput) (*ec2.ReleaseAddressOutput, error) {
	return &m.ReleaseAddressOutput, nil
}

func TestEIPAddress_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

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
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testAllocId1, testAllocId2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testAllocId2},
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
			names, err := ea.getAll(context.Background(), config.Config{
				ElasticIP: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, awsgo.StringValueSlice(names))
		})
	}

}

func TestEIPAddress_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	ea := EIPAddresses{
		Client: &mockedEIPAddresses{
			ReleaseAddressOutput: ec2.ReleaseAddressOutput{},
		},
	}

	err := ea.nukeAll([]*string{awsgo.String("alloc1")})
	require.NoError(t, err)
}
