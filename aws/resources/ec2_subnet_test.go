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

type mockEC2SubnetClient struct {
	DescribeOutput  ec2.DescribeSubnetsOutput
	DeleteOutput    ec2.DeleteSubnetOutput
	CreateTagsCalls []ec2.CreateTagsInput
}

func (m *mockEC2SubnetClient) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	return &m.DescribeOutput, nil
}

func (m *mockEC2SubnetClient) DeleteSubnet(ctx context.Context, params *ec2.DeleteSubnetInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSubnetOutput, error) {
	return &m.DeleteOutput, nil
}

func (m *mockEC2SubnetClient) CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error) {
	m.CreateTagsCalls = append(m.CreateTagsCalls, *params)
	return &ec2.CreateTagsOutput{}, nil
}

func subnetTestContext() context.Context {
	return context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
}

func TestListEC2Subnets(t *testing.T) {
	t.Parallel()

	mock := &mockEC2SubnetClient{
		DescribeOutput: ec2.DescribeSubnetsOutput{
			Subnets: []types.Subnet{
				{
					SubnetId: aws.String("subnet-001"),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("subnet1")},
					},
				},
				{
					SubnetId: aws.String("subnet-002"),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("subnet2")},
					},
				},
			},
		},
	}

	ids, err := listEC2Subnets(subnetTestContext(), mock, resource.Scope{}, config.ResourceType{}, false)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"subnet-001", "subnet-002"}, aws.ToStringSlice(ids))
}

func TestListEC2Subnets_WithFilter(t *testing.T) {
	t.Parallel()

	mock := &mockEC2SubnetClient{
		DescribeOutput: ec2.DescribeSubnetsOutput{
			Subnets: []types.Subnet{
				{
					SubnetId: aws.String("subnet-001"),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("keep-this")},
					},
				},
				{
					SubnetId: aws.String("subnet-002"),
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String("skip-this")},
					},
				},
			},
		},
	}

	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{
			NamesRegExp: []config.Expression{{RE: *regexp.MustCompile("skip-.*")}},
		},
	}

	ids, err := listEC2Subnets(subnetTestContext(), mock, resource.Scope{}, cfg, false)
	require.NoError(t, err)
	require.Equal(t, []string{"subnet-001"}, aws.ToStringSlice(ids))
}

func TestListEC2Subnets_SkipsDefaultSubnets(t *testing.T) {
	t.Parallel()

	mock := &mockEC2SubnetClient{
		DescribeOutput: ec2.DescribeSubnetsOutput{
			Subnets: []types.Subnet{
				{SubnetId: aws.String("subnet-default"), DefaultForAz: aws.Bool(true)},
				{SubnetId: aws.String("subnet-custom"), DefaultForAz: aws.Bool(false)},
				{SubnetId: aws.String("subnet-nil")}, // DefaultForAz unset (nil)
			},
		},
	}

	// defaultOnly=false: default subnets are skipped, non-default and nil are kept
	ids, err := listEC2Subnets(subnetTestContext(), mock, resource.Scope{}, config.ResourceType{}, false)
	require.NoError(t, err)
	require.Equal(t, []string{"subnet-custom", "subnet-nil"}, aws.ToStringSlice(ids))
}

// Regression test for https://github.com/gruntwork-io/cloud-nuke/issues/1153:
// subnets without a first-seen tag must get one stamped during listing, so a
// time filter (--older-than) can pick them up on a later run instead of
// excluding them forever.
func TestListEC2Subnets_StampsFirstSeenTag(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	mock := &mockEC2SubnetClient{
		DescribeOutput: ec2.DescribeSubnetsOutput{
			Subnets: []types.Subnet{
				{SubnetId: aws.String("subnet-untagged")},
				{
					SubnetId: aws.String("subnet-old"),
					Tags: []types.Tag{
						{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(now.Add(-24 * time.Hour)))},
					},
				},
			},
		},
	}

	// Equivalent of --older-than 12h: exclude anything first seen after now-12h.
	excludeAfter := now.Add(-12 * time.Hour)
	cfg := config.ResourceType{
		ExcludeRule: config.FilterRule{TimeAfter: &excludeAfter},
	}

	ids, err := listEC2Subnets(subnetTestContext(), mock, resource.Scope{}, cfg, false)
	require.NoError(t, err)

	// The subnet first seen 24h ago passes the 12h filter; the untagged one is
	// first seen "now", so it is excluded this run but tagged for future runs.
	require.Equal(t, []string{"subnet-old"}, aws.ToStringSlice(ids))
	require.Len(t, mock.CreateTagsCalls, 1)
	require.Equal(t, []string{"subnet-untagged"}, mock.CreateTagsCalls[0].Resources)
	require.Equal(t, util.FirstSeenTagKey, aws.ToString(mock.CreateTagsCalls[0].Tags[0].Key))
}

func TestListEC2Subnets_ExcludeFirstSeenTag(t *testing.T) {
	t.Parallel()

	mock := &mockEC2SubnetClient{
		DescribeOutput: ec2.DescribeSubnetsOutput{
			Subnets: []types.Subnet{
				{SubnetId: aws.String("subnet-untagged")},
			},
		},
	}

	// With --exclude-first-seen, listing must not write any tags.
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, true)
	ids, err := listEC2Subnets(ctx, mock, resource.Scope{}, config.ResourceType{}, false)
	require.NoError(t, err)
	require.Equal(t, []string{"subnet-untagged"}, aws.ToStringSlice(ids))
	require.Empty(t, mock.CreateTagsCalls)
}

func TestDeleteSubnet(t *testing.T) {
	t.Parallel()

	mock := &mockEC2SubnetClient{}
	err := deleteSubnet(context.Background(), mock, aws.String("subnet-test"))
	require.NoError(t, err)
}
