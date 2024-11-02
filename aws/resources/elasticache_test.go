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
	"github.com/stretchr/testify/require"
)

type mockedElasticache struct {
	ElasticachesAPI
	DescribeReplicationGroupsOutput elasticache.DescribeReplicationGroupsOutput
	DescribeCacheClustersOutput     elasticache.DescribeCacheClustersOutput
	DeleteCacheClusterOutput        elasticache.DeleteCacheClusterOutput
	DeleteReplicationGroupOutput    elasticache.DeleteReplicationGroupOutput
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

func TestElasticache_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()
	testName1 := "test-name-1"
	testName2 := "test-name-2"
	ec := Elasticaches{
		Client: mockedElasticache{
			DescribeReplicationGroupsOutput: elasticache.DescribeReplicationGroupsOutput{
				ReplicationGroups: []types.ReplicationGroup{
					{
						ReplicationGroupId:         aws.String(testName1),
						ReplicationGroupCreateTime: aws.Time(now),
					},
					{
						ReplicationGroupId:         aws.String(testName2),
						ReplicationGroupCreateTime: aws.Time(now.Add(1)),
					},
				},
			},
			DescribeCacheClustersOutput: elasticache.DescribeCacheClustersOutput{
				CacheClusters: []types.CacheCluster{},
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
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := ec.getAll(context.Background(), config.Config{
				Elasticache: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestElasticache_NukeAll(t *testing.T) {
	t.Parallel()

	ec := Elasticaches{
		BaseAwsResource: BaseAwsResource{
			Context: context.Background(),
		},
		Client: mockedElasticache{
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
		},
	}

	err := ec.nukeAll(aws.StringSlice([]string{"test-name-1"}))
	require.NoError(t, err)
}
