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

type mockEIPAddressesClient struct {
	DescribeAddressesOutput ec2.DescribeAddressesOutput
	ReleaseAddressOutput    ec2.ReleaseAddressOutput
}

func (m *mockEIPAddressesClient) DescribeAddresses(ctx context.Context, params *ec2.DescribeAddressesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error) {
	return &m.DescribeAddressesOutput, nil
}

func (m *mockEIPAddressesClient) ReleaseAddress(ctx context.Context, params *ec2.ReleaseAddressInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseAddressOutput, error) {
	return &m.ReleaseAddressOutput, nil
}

func (m *mockEIPAddressesClient) CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error) {
	return &ec2.CreateTagsOutput{}, nil
}

func TestListEIPAddresses(t *testing.T) {
	t.Parallel()

	now := time.Now()
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	mock := &mockEIPAddressesClient{
		DescribeAddressesOutput: ec2.DescribeAddressesOutput{
			Addresses: []types.Address{
				{
					AllocationId: aws.String("eipalloc-1"),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("test-eip1")},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
					},
				},
				{
					AllocationId: aws.String("eipalloc-2"),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("test-eip2")},
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now))},
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
			expected:  []string{"eipalloc-1", "eipalloc-2"},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("test-eip1")}},
				},
			},
			expected: []string{"eipalloc-2"},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				},
			},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, err := listEIPAddresses(ctx, mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(ids))
		})
	}
}

func TestReleaseEIPAddress(t *testing.T) {
	t.Parallel()

	mock := &mockEIPAddressesClient{}
	err := releaseEIPAddress(context.Background(), mock, aws.String("eipalloc-1"))
	require.NoError(t, err)
}
