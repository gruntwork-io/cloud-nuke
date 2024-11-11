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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedElasticCacheServerlessService struct {
	ElasticCacheServerlessAPI
	DeleteServerlessCacheOutput    elasticache.DeleteServerlessCacheOutput
	DescribeServerlessCachesOutput elasticache.DescribeServerlessCachesOutput
}

func (m mockedElasticCacheServerlessService) DeleteServerlessCache(ctx context.Context, params *elasticache.DeleteServerlessCacheInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteServerlessCacheOutput, error) {
	return &m.DeleteServerlessCacheOutput, nil
}

func (m mockedElasticCacheServerlessService) DescribeServerlessCaches(ctx context.Context, params *elasticache.DescribeServerlessCachesInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeServerlessCachesOutput, error) {
	return &m.DescribeServerlessCachesOutput, nil
}

func Test_ElasticCacheServerless_NukeAll(t *testing.T) {
	t.Parallel()

	clusterName := "test-workspace-1"
	service := ElasticCacheServerless{
		Client: mockedElasticCacheServerlessService{
			DeleteServerlessCacheOutput: elasticache.DeleteServerlessCacheOutput{},
		},
	}

	err := service.nukeAll([]*string{&clusterName})
	assert.NoError(t, err)
}

func Test_ElasticCacheServerless_GetAll(t *testing.T) {
	t.Parallel()

	now := time.Now()
	cluster1 := "test-workspace-1"
	cluster2 := "test-workspace-2"

	service := ElasticCacheServerless{
		Client: mockedElasticCacheServerlessService{
			DescribeServerlessCachesOutput: elasticache.DescribeServerlessCachesOutput{
				ServerlessCaches: []types.ServerlessCache{
					{
						ARN:        aws.String("arn::region::" + cluster1),
						CreateTime: &now,
						Status:     aws.String("available"),
					},
					{
						ARN:        aws.String("arn::region::" + cluster2),
						CreateTime: aws.Time(now.Add(1 * time.Hour)),
						Status:     aws.String("available"),
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
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			workspaces, err := service.getAll(
				context.Background(),
				config.Config{ElasticCacheServerless: tc.configObj},
			)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(workspaces))
		})
	}
}
