package resources

import (
	"context"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockDBSubnetGroupsClient struct {
	DescribeDBSubnetGroupsOutput rds.DescribeDBSubnetGroupsOutput
	DeleteDBSubnetGroupOutput    rds.DeleteDBSubnetGroupOutput
	DeleteDBSubnetGroupError     error
	ListTagsOutput               rds.ListTagsForResourceOutput
}

func (m *mockDBSubnetGroupsClient) DescribeDBSubnetGroups(ctx context.Context, params *rds.DescribeDBSubnetGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSubnetGroupsOutput, error) {
	return &m.DescribeDBSubnetGroupsOutput, nil
}

func (m *mockDBSubnetGroupsClient) DeleteDBSubnetGroup(ctx context.Context, params *rds.DeleteDBSubnetGroupInput, optFns ...func(*rds.Options)) (*rds.DeleteDBSubnetGroupOutput, error) {
	return &m.DeleteDBSubnetGroupOutput, m.DeleteDBSubnetGroupError
}

func (m *mockDBSubnetGroupsClient) ListTagsForResource(ctx context.Context, params *rds.ListTagsForResourceInput, optFns ...func(*rds.Options)) (*rds.ListTagsForResourceOutput, error) {
	if m.ListTagsOutput.TagList != nil {
		return &m.ListTagsOutput, nil
	}
	return &rds.ListTagsForResourceOutput{
		TagList: []types.Tag{
			{Key: aws.String("env"), Value: aws.String("test")},
		},
	}, nil
}

func TestListDBSubnetGroups(t *testing.T) {
	t.Parallel()

	testName1 := "test-db-subnet-group1"
	testName2 := "test-db-subnet-group2"

	mock := &mockDBSubnetGroupsClient{
		DescribeDBSubnetGroupsOutput: rds.DescribeDBSubnetGroupsOutput{
			DBSubnetGroups: []types.DBSubnetGroup{
				{
					DBSubnetGroupName: aws.String(testName1),
					DBSubnetGroupArn:  aws.String("arn:" + testName1),
				},
				{
					DBSubnetGroupName: aws.String(testName2),
					DBSubnetGroupArn:  aws.String("arn:" + testName2),
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
			names, err := listDBSubnetGroups(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteDBSubnetGroup(t *testing.T) {
	t.Parallel()

	mock := &mockDBSubnetGroupsClient{
		DeleteDBSubnetGroupOutput: rds.DeleteDBSubnetGroupOutput{},
	}

	err := deleteDBSubnetGroup(context.Background(), mock, aws.String("test"))
	require.NoError(t, err)
}
