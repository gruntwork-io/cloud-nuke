package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/elasticache/elasticacheiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedElasticache struct {
	elasticacheiface.ElastiCacheAPI
	DescribeReplicationGroupsOutput elasticache.DescribeReplicationGroupsOutput
	DescribeCacheClustersOutput     elasticache.DescribeCacheClustersOutput
	DeleteReplicationGroupOutput    elasticache.DeleteReplicationGroupOutput
}

func (m mockedElasticache) DescribeCacheClustersWithContext(_ aws.Context, input *elasticache.DescribeCacheClustersInput, _ ...request.Option) (*elasticache.DescribeCacheClustersOutput, error) {
	return &m.DescribeCacheClustersOutput, nil
}

func (m mockedElasticache) DescribeReplicationGroupsWithContext(_ aws.Context, input *elasticache.DescribeReplicationGroupsInput, _ ...request.Option) (*elasticache.DescribeReplicationGroupsOutput, error) {
	return &m.DescribeReplicationGroupsOutput, nil
}

func (m mockedElasticache) DeleteReplicationGroupWithContext(_ aws.Context, input *elasticache.DeleteReplicationGroupInput, _ ...request.Option) (*elasticache.DeleteReplicationGroupOutput, error) {
	return &m.DeleteReplicationGroupOutput, nil
}

func (m mockedElasticache) WaitUntilReplicationGroupDeletedWithContext(_ aws.Context, input *elasticache.DescribeReplicationGroupsInput, _ ...request.WaiterOption) error {
	return nil
}

func TestElasticache_GetAll(t *testing.T) {

	t.Parallel()

	now := time.Now()
	testName1 := "test-name-1"
	testName2 := "test-name-2"
	ec := Elasticaches{
		Client: mockedElasticache{
			DescribeReplicationGroupsOutput: elasticache.DescribeReplicationGroupsOutput{
				ReplicationGroups: []*elasticache.ReplicationGroup{
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
				CacheClusters: []*elasticache.CacheCluster{},
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
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}
}

func TestElasticache_NukeAll(t *testing.T) {

	t.Parallel()

	now := time.Now()
	ec := Elasticaches{
		Client: mockedElasticache{
			DescribeReplicationGroupsOutput: elasticache.DescribeReplicationGroupsOutput{
				ReplicationGroups: []*elasticache.ReplicationGroup{
					{
						ReplicationGroupId:         aws.String("test-name-1"),
						ReplicationGroupCreateTime: aws.Time(now),
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
