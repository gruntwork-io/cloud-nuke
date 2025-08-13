package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedEC2Cluster struct {
	ECSClustersAPI
	DescribeClustersOutput    ecs.DescribeClustersOutput
	DeleteClusterOutput       ecs.DeleteClusterOutput
	ListClustersOutput        ecs.ListClustersOutput
	ListTagsForResourceOutput ecs.ListTagsForResourceOutput
	ListTasksOutput           ecs.ListTasksOutput
	StopTaskOutput            ecs.StopTaskOutput
	TagResourceOutput         ecs.TagResourceOutput
}

func (m mockedEC2Cluster) DescribeClusters(ctx context.Context, params *ecs.DescribeClustersInput, optFns ...func(*ecs.Options)) (*ecs.DescribeClustersOutput, error) {
	return &m.DescribeClustersOutput, nil
}

func (m mockedEC2Cluster) DeleteCluster(ctx context.Context, params *ecs.DeleteClusterInput, optFns ...func(*ecs.Options)) (*ecs.DeleteClusterOutput, error) {
	return &m.DeleteClusterOutput, nil
}

func (m mockedEC2Cluster) ListClusters(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error) {
	return &m.ListClustersOutput, nil
}

func (m mockedEC2Cluster) ListTagsForResource(ctx context.Context, params *ecs.ListTagsForResourceInput, optFns ...func(*ecs.Options)) (*ecs.ListTagsForResourceOutput, error) {
	return &m.ListTagsForResourceOutput, nil
}

func (m mockedEC2Cluster) ListTasks(ctx context.Context, params *ecs.ListTasksInput, optFns ...func(*ecs.Options)) (*ecs.ListTasksOutput, error) {
	return &m.ListTasksOutput, nil
}

func (m mockedEC2Cluster) StopTask(ctx context.Context, params *ecs.StopTaskInput, optFns ...func(*ecs.Options)) (*ecs.StopTaskOutput, error) {
	return &m.StopTaskOutput, nil
}

func (m mockedEC2Cluster) TagResource(ctx context.Context, params *ecs.TagResourceInput, optFns ...func(*ecs.Options)) (*ecs.TagResourceOutput, error) {
	return &m.TagResourceOutput, nil
}

func TestEC2Cluster_GetAll(t *testing.T) {
	t.Parallel()
	// Set excludeFirstSeenTag to false for testing
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	testArn1 := "arn:aws:ecs:us-east-1:123456789012:cluster/cluster1"
	testArn2 := "arn:aws:ecs:us-east-1:123456789012:cluster/cluster2"
	testName1 := "cluster1"
	testName2 := "cluster2"
	now := time.Now()
	ec := ECSClusters{
		Client: mockedEC2Cluster{
			ListClustersOutput: ecs.ListClustersOutput{
				ClusterArns: []string{
					testArn1,
					testArn2,
				},
			},

			DescribeClustersOutput: ecs.DescribeClustersOutput{
				Clusters: []types.Cluster{
					{
						ClusterArn:  aws.String(testArn1),
						Status:      aws.String("ACTIVE"),
						ClusterName: aws.String(testName1),
					},
					{
						ClusterArn:  aws.String(testArn2),
						Status:      aws.String("ACTIVE"),
						ClusterName: aws.String(testName2),
					},
				},
			},

			ListTagsForResourceOutput: ecs.ListTagsForResourceOutput{
				Tags: []types.Tag{
					{
						Key:   aws.String(util.FirstSeenTagKey),
						Value: aws.String(util.FormatTimestamp(now)),
					},
				},
			},
			ListTasksOutput: ecs.ListTasksOutput{
				TaskArns: []string{
					"task-arn-001",
					"task-arn-002",
				},
			},
		},
	}

	tests := map[string]struct {
		ctx       context.Context
		configObj config.ResourceType
		expected  []string
	}{
		"emptyFilter": {
			ctx:       ctx,
			configObj: config.ResourceType{},
			expected:  []string{testArn1, testArn2},
		},
		"nameExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testArn2},
		},
		"timeAfterExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				}},
			expected: []string{},
		},
		"nameInclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}},
				},
			},
			expected: []string{testArn1},
		},
		"timeBeforeExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					TimeBefore: aws.Time(now.Add(1 * time.Hour)),
				},
			},
			expected: []string{},
		},
		"timeAfterInclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					TimeAfter: aws.Time(now.Add(-1 * time.Hour)),
				},
			},
			expected: []string{testArn1, testArn2},
		},
		"timeBeforeInclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					TimeBefore: aws.Time(now.Add(1 * time.Hour)),
				},
			},
			expected: []string{testArn1, testArn2},
		},
		"combinedIncludeExcludeFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile("cluster.*"),
					}},
				},
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName2),
					}},
				},
			},
			expected: []string{testArn1},
		},
		"excludeFirstSeenTag": {
			ctx:       context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, true),
			configObj: config.ResourceType{},
			expected:  []string{testArn1, testArn2},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := ec.getAll(tc.ctx, config.Config{
				ECSCluster: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestEC2Cluster_GetAll_InactiveClusters(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	testArn1 := "arn:aws:ecs:us-east-1:123456789012:cluster/inactive1"
	testArn2 := "arn:aws:ecs:us-east-1:123456789012:cluster/active1"
	now := time.Now()

	ec := ECSClusters{
		Client: mockedEC2Cluster{
			ListClustersOutput: ecs.ListClustersOutput{
				ClusterArns: []string{testArn1, testArn2},
			},
			DescribeClustersOutput: ecs.DescribeClustersOutput{
				Clusters: []types.Cluster{
					{
						ClusterArn:  aws.String(testArn1),
						Status:      aws.String("INACTIVE"),
						ClusterName: aws.String("inactive1"),
					},
					{
						ClusterArn:  aws.String(testArn2),
						Status:      aws.String("ACTIVE"),
						ClusterName: aws.String("active1"),
					},
				},
			},
			ListTagsForResourceOutput: ecs.ListTagsForResourceOutput{
				Tags: []types.Tag{
					{
						Key:   aws.String(util.FirstSeenTagKey),
						Value: aws.String(util.FormatTimestamp(now)),
					},
				},
			},
		},
	}

	names, err := ec.getAll(ctx, config.Config{ECSCluster: config.ResourceType{}})
	require.NoError(t, err)
	// Only active cluster should be returned
	require.Equal(t, []string{testArn2}, aws.ToStringSlice(names))
}

func TestEC2Cluster_GetAll_NoFirstSeenTag(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	testArn := "arn:aws:ecs:us-east-1:123456789012:cluster/new-cluster"
	ec := ECSClusters{
		Client: mockedEC2Cluster{
			ListClustersOutput: ecs.ListClustersOutput{
				ClusterArns: []string{testArn},
			},
			DescribeClustersOutput: ecs.DescribeClustersOutput{
				Clusters: []types.Cluster{
					{
						ClusterArn:  aws.String(testArn),
						Status:      aws.String("ACTIVE"),
						ClusterName: aws.String("new-cluster"),
					},
				},
			},
			ListTagsForResourceOutput: ecs.ListTagsForResourceOutput{
				Tags: []types.Tag{}, // No tags
			},
			TagResourceOutput: ecs.TagResourceOutput{},
		},
	}

	names, err := ec.getAll(ctx, config.Config{ECSCluster: config.ResourceType{}})
	require.NoError(t, err)
	// Should return empty since cluster gets tagged but not included until next run
	require.Empty(t, names)
}

func TestEC2Cluster_GetAll_EmptyList(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	ec := ECSClusters{
		Client: mockedEC2Cluster{
			ListClustersOutput:        ecs.ListClustersOutput{ClusterArns: []string{}},
			DescribeClustersOutput:    ecs.DescribeClustersOutput{Clusters: []types.Cluster{}},
			ListTagsForResourceOutput: ecs.ListTagsForResourceOutput{Tags: []types.Tag{}},
		},
	}

	names, err := ec.getAll(ctx, config.Config{ECSCluster: config.ResourceType{}})
	require.NoError(t, err)
	require.Empty(t, names)
}

func TestEC2Cluster_NukeAll(t *testing.T) {
	t.Parallel()
	ec := ECSClusters{
		Client: mockedEC2Cluster{
			DeleteClusterOutput: ecs.DeleteClusterOutput{},
			ListTasksOutput:     ecs.ListTasksOutput{TaskArns: []string{}},
		},
	}

	err := ec.nukeAll([]*string{aws.String("arn:aws:ecs:us-east-1:123456789012:cluster/cluster1")})
	require.NoError(t, err)
}

func TestEC2Cluster_NukeAll_EmptyList(t *testing.T) {
	t.Parallel()
	ec := ECSClusters{
		Client: mockedEC2Cluster{},
	}

	err := ec.nukeAll([]*string{})
	require.NoError(t, err)
}

func TestEC2ClusterWithTasks_NukeAll(t *testing.T) {
	t.Parallel()
	ec := ECSClusters{
		Client: mockedEC2Cluster{
			DeleteClusterOutput: ecs.DeleteClusterOutput{},
			ListTasksOutput: ecs.ListTasksOutput{
				TaskArns: []string{
					"task-arn-001",
					"task-arn-002",
				},
			},
			StopTaskOutput: ecs.StopTaskOutput{},
		},
	}

	err := ec.nukeAll([]*string{aws.String("arn:aws:ecs:us-east-1:123456789012:cluster/cluster1")})
	require.NoError(t, err)
}

func TestEC2Cluster_GetAll_MultipleRegexPatterns(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	testArn1 := "arn:aws:ecs:us-east-1:123456789012:cluster/prod-cluster"
	testArn2 := "arn:aws:ecs:us-east-1:123456789012:cluster/dev-cluster"
	testArn3 := "arn:aws:ecs:us-east-1:123456789012:cluster/test-service"
	now := time.Now()

	ec := ECSClusters{
		Client: mockedEC2Cluster{
			ListClustersOutput: ecs.ListClustersOutput{
				ClusterArns: []string{testArn1, testArn2, testArn3},
			},
			DescribeClustersOutput: ecs.DescribeClustersOutput{
				Clusters: []types.Cluster{
					{
						ClusterArn:  aws.String(testArn1),
						Status:      aws.String("ACTIVE"),
						ClusterName: aws.String("prod-cluster"),
					},
					{
						ClusterArn:  aws.String(testArn2),
						Status:      aws.String("ACTIVE"),
						ClusterName: aws.String("dev-cluster"),
					},
					{
						ClusterArn:  aws.String(testArn3),
						Status:      aws.String("ACTIVE"),
						ClusterName: aws.String("test-service"),
					},
				},
			},
			ListTagsForResourceOutput: ecs.ListTagsForResourceOutput{
				Tags: []types.Tag{
					{
						Key:   aws.String(util.FirstSeenTagKey),
						Value: aws.String(util.FormatTimestamp(now)),
					},
				},
			},
		},
	}

	tests := map[string]struct {
		configObj config.ResourceType
		expected  []string
	}{
		"includeMultiplePatterns": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{
						{RE: *regexp.MustCompile(".*-cluster$")},
						{RE: *regexp.MustCompile("test-.*")},
					},
				},
			},
			expected: []string{testArn1, testArn2, testArn3},
		},
		"excludeMultiplePatterns": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{
						{RE: *regexp.MustCompile("prod-.*")},
						{RE: *regexp.MustCompile(".*-service$")},
					},
				},
			},
			expected: []string{testArn2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := ec.getAll(ctx, config.Config{ECSCluster: tc.configObj})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}
