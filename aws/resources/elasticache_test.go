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

type mockedElasticache struct {
	ElasticachesAPI
	DescribeReplicationGroupsOutput elasticache.DescribeReplicationGroupsOutput
	DescribeCacheClustersOutput     elasticache.DescribeCacheClustersOutput
	DeleteCacheClusterOutput        elasticache.DeleteCacheClusterOutput
	DeleteReplicationGroupOutput    elasticache.DeleteReplicationGroupOutput
	TagsByARN                       map[string][]types.Tag
}

func (m mockedElasticache) DescribeReplicationGroups(ctx context.Context, params *elasticache.DescribeReplicationGroupsInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeReplicationGroupsOutput, error) {
	return &m.DescribeReplicationGroupsOutput, nil
}

func (m mockedElasticache) DescribeCacheClusters(ctx context.Context, params *elasticache.DescribeCacheClustersInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheClustersOutput, error) {
	return &m.DescribeCacheClustersOutput, nil
}

func (m mockedElasticache) DeleteCacheCluster(ctx context.Context, params *elasticache.DeleteCacheClusterInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteCacheClusterOutput, error) {
	return &m.DeleteCacheClusterOutput, nil
}

func (m mockedElasticache) DeleteReplicationGroup(ctx context.Context, params *elasticache.DeleteReplicationGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteReplicationGroupOutput, error) {
	return &m.DeleteReplicationGroupOutput, nil
}

func (m mockedElasticache) ListTagsForResource(ctx context.Context, params *elasticache.ListTagsForResourceInput, optFns ...func(*elasticache.Options)) (*elasticache.ListTagsForResourceOutput, error) {
	return &elasticache.ListTagsForResourceOutput{TagList: m.TagsByARN[aws.ToString(params.ResourceName)]}, nil
}

func TestElasticaches_ResourceName(t *testing.T) {
	r := NewElasticaches()
	require.Equal(t, "elasticache", r.ResourceName())
}

func TestElasticaches_MaxBatchSize(t *testing.T) {
	r := NewElasticaches()
	require.Equal(t, DefaultBatchSize, r.MaxBatchSize())
}

func TestListElasticaches(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testName1 := "test-name-1"
	testName2 := "test-name-2"
	testArn1 := "arn:aws:elasticache:us-east-1:123456789:replicationgroup:" + testName1
	testArn2 := "arn:aws:elasticache:us-east-1:123456789:replicationgroup:" + testName2

	mock := mockedElasticache{
		DescribeReplicationGroupsOutput: elasticache.DescribeReplicationGroupsOutput{
			ReplicationGroups: []types.ReplicationGroup{
				{
					ReplicationGroupId:         aws.String(testName1),
					ARN:                        aws.String(testArn1),
					ReplicationGroupCreateTime: aws.Time(now),
				},
				{
					ReplicationGroupId:         aws.String(testName2),
					ARN:                        aws.String(testArn2),
					ReplicationGroupCreateTime: aws.Time(now.Add(1)),
				},
			},
		},
		DescribeCacheClustersOutput: elasticache.DescribeCacheClustersOutput{
			CacheClusters: []types.CacheCluster{},
		},
		TagsByARN: map[string][]types.Tag{
			testArn1: {{Key: aws.String("env"), Value: aws.String("prod")}},
			testArn2: {{Key: aws.String("env"), Value: aws.String("dev")}},
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
					}}},
			},
			expected: []string{testName2},
		},
		"timeAfterExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now),
				}},
			expected: []string{testName1},
		},
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"env": {RE: *regexp.MustCompile("^prod$")},
					},
				},
			},
			expected: []string{testName1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listElasticaches(context.Background(), mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestDeleteElasticacheCluster(t *testing.T) {
	t.Parallel()

	mock := mockedElasticache{
		DescribeReplicationGroupsOutput: elasticache.DescribeReplicationGroupsOutput{
			ReplicationGroups: []types.ReplicationGroup{
				{
					ReplicationGroupId:         aws.String("test-name-1"),
					ReplicationGroupCreateTime: aws.Time(time.Now()),
					Status:                     aws.String("deleted"),
				},
			},
		},
		DescribeCacheClustersOutput:  elasticache.DescribeCacheClustersOutput{},
		DeleteReplicationGroupOutput: elasticache.DeleteReplicationGroupOutput{},
	}

	err := deleteElasticacheCluster(context.Background(), mock, aws.String("test-name-1"))
	require.NoError(t, err)
}
