package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockDBSubnetGroupsClient struct {
	DescribeDBSubnetGroupsOutput rds.DescribeDBSubnetGroupsOutput
	DeleteDBSubnetGroupOutput    rds.DeleteDBSubnetGroupOutput
	DeleteDBSubnetGroupError     error
	ListTagsOutput               rds.ListTagsForResourceOutput
	// TagsByArn overrides ListTagsOutput for the given subnet group ARNs.
	TagsByArn map[string][]types.Tag
	// TaggedArns records the ARNs passed to AddTagsToResource.
	TaggedArns []string
}

func (m *mockDBSubnetGroupsClient) AddTagsToResource(ctx context.Context, params *rds.AddTagsToResourceInput, optFns ...func(*rds.Options)) (*rds.AddTagsToResourceOutput, error) {
	m.TaggedArns = append(m.TaggedArns, aws.ToString(params.ResourceName))
	return &rds.AddTagsToResourceOutput{}, nil
}

func (m *mockDBSubnetGroupsClient) DescribeDBSubnetGroups(ctx context.Context, params *rds.DescribeDBSubnetGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSubnetGroupsOutput, error) {
	return &m.DescribeDBSubnetGroupsOutput, nil
}

func (m *mockDBSubnetGroupsClient) DeleteDBSubnetGroup(ctx context.Context, params *rds.DeleteDBSubnetGroupInput, optFns ...func(*rds.Options)) (*rds.DeleteDBSubnetGroupOutput, error) {
	return &m.DeleteDBSubnetGroupOutput, m.DeleteDBSubnetGroupError
}

func (m *mockDBSubnetGroupsClient) ListTagsForResource(ctx context.Context, params *rds.ListTagsForResourceInput, optFns ...func(*rds.Options)) (*rds.ListTagsForResourceOutput, error) {
	if tags, ok := m.TagsByArn[aws.ToString(params.ResourceName)]; ok {
		return &rds.ListTagsForResourceOutput{TagList: tags}, nil
	}
	if m.ListTagsOutput.TagList != nil {
		return &m.ListTagsOutput, nil
	}
	return &rds.ListTagsForResourceOutput{
		TagList: []types.Tag{
			{Key: aws.String("env"), Value: aws.String("test")},
		},
	}, nil
}

func dbSubnetGroup(name string) types.DBSubnetGroup {
	return types.DBSubnetGroup{
		DBSubnetGroupName: aws.String(name),
		DBSubnetGroupArn:  aws.String("arn:" + name),
	}
}

func firstSeenRDSTags(t time.Time) []types.Tag {
	return []types.Tag{{Key: aws.String(util.FirstSeenTagKey), Value: aws.String(util.FormatTimestamp(t))}}
}

func TestListDBSubnetGroups(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
	testName1 := "test-db-subnet-group1"
	testName2 := "test-db-subnet-group2"

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{testName1, testName2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}},
				},
			},
			expected: []string{testName2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mock := &mockDBSubnetGroupsClient{
				DescribeDBSubnetGroupsOutput: rds.DescribeDBSubnetGroupsOutput{
					DBSubnetGroups: []types.DBSubnetGroup{dbSubnetGroup(testName1), dbSubnetGroup(testName2)},
				},
			}
			names, err := listDBSubnetGroups(ctx, mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestListDBSubnetGroups_SkipsDefault(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
	mock := &mockDBSubnetGroupsClient{
		DescribeDBSubnetGroupsOutput: rds.DescribeDBSubnetGroupsOutput{
			DBSubnetGroups: []types.DBSubnetGroup{dbSubnetGroup("default"), dbSubnetGroup("custom")},
		},
	}

	names, err := listDBSubnetGroups(ctx, mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.Equal(t, []string{"custom"}, aws.ToStringSlice(names))
	require.Equal(t, []string{"arn:custom"}, mock.TaggedArns)
}

func TestListDBSubnetGroups_FirstSeenTimeFilter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	olderThanOneHour := config.ResourceType{
		ExcludeRule: config.FilterRule{TimeAfter: aws.Time(now.Add(-1 * time.Hour))},
	}

	mock := &mockDBSubnetGroupsClient{
		DescribeDBSubnetGroupsOutput: rds.DescribeDBSubnetGroupsOutput{
			DBSubnetGroups: []types.DBSubnetGroup{dbSubnetGroup("untagged"), dbSubnetGroup("old"), dbSubnetGroup("recent")},
		},
		TagsByArn: map[string][]types.Tag{
			"arn:untagged": {},
			"arn:old":      firstSeenRDSTags(now.Add(-2 * time.Hour)),
			"arn:recent":   firstSeenRDSTags(now.Add(-10 * time.Minute)),
		},
	}

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)
	names, err := listDBSubnetGroups(ctx, mock, resource.Scope{}, olderThanOneHour)
	require.NoError(t, err)
	// The untagged group is tagged on this run and becomes eligible on a later one.
	require.Equal(t, []string{"old"}, aws.ToStringSlice(names))
	require.Equal(t, []string{"arn:untagged"}, mock.TaggedArns)
}

func TestListDBSubnetGroups_ExcludeFirstSeenTag(t *testing.T) {
	t.Parallel()

	mock := &mockDBSubnetGroupsClient{
		DescribeDBSubnetGroupsOutput: rds.DescribeDBSubnetGroupsOutput{
			DBSubnetGroups: []types.DBSubnetGroup{dbSubnetGroup("untagged")},
		},
		TagsByArn: map[string][]types.Tag{"arn:untagged": {}},
	}

	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, true)
	names, err := listDBSubnetGroups(ctx, mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.Equal(t, []string{"untagged"}, aws.ToStringSlice(names))
	require.Empty(t, mock.TaggedArns)
}

func TestDeleteDBSubnetGroup(t *testing.T) {
	t.Parallel()

	mock := &mockDBSubnetGroupsClient{
		DeleteDBSubnetGroupOutput: rds.DeleteDBSubnetGroupOutput{},
	}

	err := deleteDBSubnetGroup(context.Background(), mock, aws.String("test"))
	require.NoError(t, err)
}
