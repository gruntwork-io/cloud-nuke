package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/elasticache/elasticacheiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

type mockedElasticache struct {
	elasticacheiface.ElastiCacheAPI
	DescribeReplicationGroupsOutput elasticache.DescribeReplicationGroupsOutput
	DescribeCacheClustersOutput     elasticache.DescribeCacheClustersOutput
	DeleteReplicationGroupOutput    elasticache.DeleteReplicationGroupOutput
}

func (m mockedElasticache) DescribeCacheClusters(input *elasticache.DescribeCacheClustersInput) (*elasticache.DescribeCacheClustersOutput, error) {
	return &m.DescribeCacheClustersOutput, nil
}

func (m mockedElasticache) DescribeReplicationGroups(input *elasticache.DescribeReplicationGroupsInput) (*elasticache.DescribeReplicationGroupsOutput, error) {
	return &m.DescribeReplicationGroupsOutput, nil
}

func (m mockedElasticache) DeleteReplicationGroup(input *elasticache.DeleteReplicationGroupInput) (*elasticache.DeleteReplicationGroupOutput, error) {
	return &m.DeleteReplicationGroupOutput, nil
}

func (m mockedElasticache) WaitUntilReplicationGroupDeleted(input *elasticache.DescribeReplicationGroupsInput) error {
	return nil
}

func TestElasticache_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
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
	telemetry.InitTelemetry("cloud-nuke", "")
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
