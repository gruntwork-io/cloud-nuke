package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/stretchr/testify/require"
)

type mockedElasticCacheServerlessService struct {
	DeleteServerlessCacheOutput    elasticache.DeleteServerlessCacheOutput
	DescribeServerlessCachesOutput elasticache.DescribeServerlessCachesOutput
	TagsByARN                      map[string][]types.Tag
}

func (m mockedElasticCacheServerlessService) DeleteServerlessCache(ctx context.Context, params *elasticache.DeleteServerlessCacheInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteServerlessCacheOutput, error) {
	return &m.DeleteServerlessCacheOutput, nil
}

func (m mockedElasticCacheServerlessService) DescribeServerlessCaches(ctx context.Context, params *elasticache.DescribeServerlessCachesInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeServerlessCachesOutput, error) {
	return &m.DescribeServerlessCachesOutput, nil
}

func (m mockedElasticCacheServerlessService) ListTagsForResource(ctx context.Context, params *elasticache.ListTagsForResourceInput, optFns ...func(*elasticache.Options)) (*elasticache.ListTagsForResourceOutput, error) {
	return &elasticache.ListTagsForResourceOutput{TagList: m.TagsByARN[aws.ToString(params.ResourceName)]}, nil
}

func TestElasticCacheServerless_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()
	cluster1 := "test-workspace-1"
	cluster2 := "test-workspace-2"
	arn1 := "arn::region::" + cluster1
	arn2 := "arn::region::" + cluster2

	client := mockedElasticCacheServerlessService{
		DescribeServerlessCachesOutput: elasticache.DescribeServerlessCachesOutput{
			ServerlessCaches: []types.ServerlessCache{
				{
					ARN:        aws.String(arn1),
					CreateTime: &now,
					Status:     aws.String("available"),
				},
				{
					ARN:        aws.String(arn2),
					CreateTime: aws.Time(now.Add(1 * time.Hour)),
					Status:     aws.String("available"),
				},
			},
		},
		TagsByARN: map[string][]types.Tag{
			arn1: {{Key: aws.String("env"), Value: aws.String("prod")}},
			arn2: {{Key: aws.String("env"), Value: aws.String("dev")}},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			configObj: config.ResourceType{},
			expected:  []string{cluster1, cluster2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(cluster1),
					}},
				}},
			expected: []string{cluster2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"env": {RE: *regexp.MustCompile("^prod$")},
					},
				},
			},
			expected: []string{cluster1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listElasticCacheServerless(
				context.Background(),
				client,
				resource.Scope{Region: "us-east-1"},
				tc.configObj,
			)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestElasticCacheServerless_NukeAll(t *testing.T) {
	t.Parallel()

	clusterName := "test-workspace-1"
	client := mockedElasticCacheServerlessService{
		DeleteServerlessCacheOutput: elasticache.DeleteServerlessCacheOutput{},
	}

	err := deleteElasticCacheServerless(context.Background(), client, &clusterName)
	require.NoError(t, err)
}
