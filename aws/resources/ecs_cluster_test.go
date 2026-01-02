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
	"github.com/gruntwork-io/cloud-nuke/resource"
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
	// Return different tags based on the cluster ARN
	if params.ResourceArn != nil {
		switch *params.ResourceArn {
		case "arn:aws:ecs:us-east-1:123456789012:cluster/cluster1":
			return &ecs.ListTagsForResourceOutput{
				Tags: []types.Tag{
					{
						Key:   aws.String(util.FirstSeenTagKey),
						Value: aws.String(util.FormatTimestamp(time.Now())),
					},
					{
						Key:   aws.String("Environment"),
						Value: aws.String("test"),
					},
					{
						Key:   aws.String("Team"),
						Value: aws.String("backend"),
					},
				},
			}, nil
		case "arn:aws:ecs:us-east-1:123456789012:cluster/cluster2":
			return &ecs.ListTagsForResourceOutput{
				Tags: []types.Tag{
					{
						Key:   aws.String(util.FirstSeenTagKey),
						Value: aws.String(util.FormatTimestamp(time.Now())),
					},
					{
						Key:   aws.String("Environment"),
						Value: aws.String("production"),
					},
					{
						Key:   aws.String("Team"),
						Value: aws.String("frontend"),
					},
				},
			}, nil
		}
	}
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

func TestECSClusters_ResourceName(t *testing.T) {
	r := NewECSClusters()
	require.Equal(t, "ecscluster", r.ResourceName())
}

func TestECSClusters_MaxBatchSize(t *testing.T) {
	r := NewECSClusters()
	require.Equal(t, maxBatchSize, r.MaxBatchSize())
}

func TestListECSClusters(t *testing.T) {
	t.Parallel()
	// Set excludeFirstSeenTag to false for testing
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	testArn1 := "arn:aws:ecs:us-east-1:123456789012:cluster/cluster1"
	testArn2 := "arn:aws:ecs:us-east-1:123456789012:cluster/cluster2"
	testName1 := "cluster1"
	testName2 := "cluster2"
	now := time.Now()

	mock := mockedEC2Cluster{
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
		"tagInclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"Environment": {RE: *regexp.MustCompile("test")},
					},
				},
			},
			expected: []string{testArn1},
		},
		"tagExclusionFilter": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"Environment": {RE: *regexp.MustCompile("test")},
					},
				},
			},
			expected: []string{testArn2},
		},
		"tagInclusionMultipleTagsAnd": {
			ctx: ctx,
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"Environment": {RE: *regexp.MustCompile("test")},
						"Team":        {RE: *regexp.MustCompile("backend")},
					},
					TagsOperator: "AND",
				},
			},
			expected: []string{testArn1},
		},
		"tagInclusionMultipleTagsOr": {
			ctx: ctx,
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"Environment": {RE: *regexp.MustCompile("test")},
						"Team":        {RE: *regexp.MustCompile("frontend")},
					},
					TagsOperator: "OR",
				},
			},
			expected: []string{testArn1, testArn2},
		},
		"tagExclusionMultipleTagsAnd": {
			ctx: ctx,
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{
						"Environment": {RE: *regexp.MustCompile("production")},
						"Team":        {RE: *regexp.MustCompile("frontend")},
					},
					TagsOperator: "AND",
				},
			},
			expected: []string{testArn1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := listECSClusters(tc.ctx, mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestListECSClusters_InactiveClusters(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	testArn1 := "arn:aws:ecs:us-east-1:123456789012:cluster/inactive1"
	testArn2 := "arn:aws:ecs:us-east-1:123456789012:cluster/active1"
	now := time.Now()

	mock := mockedEC2Cluster{
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
	}

	names, err := listECSClusters(ctx, mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	// Only active cluster should be returned
	require.Equal(t, []string{testArn2}, aws.ToStringSlice(names))
}

func TestListECSClusters_NoFirstSeenTag(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	testArn := "arn:aws:ecs:us-east-1:123456789012:cluster/new-cluster"
	mock := mockedEC2Cluster{
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
	}

	names, err := listECSClusters(ctx, mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	// Should return empty since cluster gets tagged but not included until next run
	require.Empty(t, names)
}

func TestListECSClusters_EmptyList(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	mock := mockedEC2Cluster{
		ListClustersOutput:        ecs.ListClustersOutput{ClusterArns: []string{}},
		DescribeClustersOutput:    ecs.DescribeClustersOutput{Clusters: []types.Cluster{}},
		ListTagsForResourceOutput: ecs.ListTagsForResourceOutput{Tags: []types.Tag{}},
	}

	names, err := listECSClusters(ctx, mock, resource.Scope{}, config.ResourceType{})
	require.NoError(t, err)
	require.Empty(t, names)
}

func TestDeleteECSCluster(t *testing.T) {
	t.Parallel()

	mock := mockedEC2Cluster{
		DeleteClusterOutput: ecs.DeleteClusterOutput{},
	}

	err := deleteECSCluster(context.Background(), mock, aws.String("arn:aws:ecs:us-east-1:123456789012:cluster/cluster1"))
	require.NoError(t, err)
}

func TestStopClusterRunningTasks(t *testing.T) {
	t.Parallel()

	mock := mockedEC2Cluster{
		ListTasksOutput: ecs.ListTasksOutput{TaskArns: []string{}},
	}

	err := stopClusterRunningTasks(context.Background(), mock, aws.String("arn:aws:ecs:us-east-1:123456789012:cluster/cluster1"))
	require.NoError(t, err)
}

func TestStopClusterRunningTasksWithTasks(t *testing.T) {
	t.Parallel()

	mock := mockedEC2Cluster{
		ListTasksOutput: ecs.ListTasksOutput{
			TaskArns: []string{
				"task-arn-001",
				"task-arn-002",
			},
		},
		StopTaskOutput: ecs.StopTaskOutput{},
	}

	err := stopClusterRunningTasks(context.Background(), mock, aws.String("arn:aws:ecs:us-east-1:123456789012:cluster/cluster1"))
	require.NoError(t, err)
}

func TestECSClustersMultiStepDeleter(t *testing.T) {
	t.Parallel()

	mock := mockedEC2Cluster{
		DeleteClusterOutput: ecs.DeleteClusterOutput{},
		ListTasksOutput: ecs.ListTasksOutput{
			TaskArns: []string{
				"task-arn-001",
				"task-arn-002",
			},
		},
		StopTaskOutput: ecs.StopTaskOutput{},
	}

	nuker := resource.MultiStepDeleter(stopClusterRunningTasks, deleteECSCluster)
	results := nuker(context.Background(), mock, resource.Scope{Region: "us-east-1"}, "ecscluster", []*string{aws.String("arn:aws:ecs:us-east-1:123456789012:cluster/cluster1")})
	require.Len(t, results, 1)
	for _, result := range results {
		require.NoError(t, result.Error)
	}
}

func TestECSClustersMultiStepDeleter_EmptyList(t *testing.T) {
	t.Parallel()

	mock := mockedEC2Cluster{}

	nuker := resource.MultiStepDeleter(stopClusterRunningTasks, deleteECSCluster)
	results := nuker(context.Background(), mock, resource.Scope{Region: "us-east-1"}, "ecscluster", []*string{})
	require.Len(t, results, 0)
}

func TestListECSClusters_MultipleRegexPatterns(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), util.ExcludeFirstSeenTagKey, false)

	testArn1 := "arn:aws:ecs:us-east-1:123456789012:cluster/prod-cluster"
	testArn2 := "arn:aws:ecs:us-east-1:123456789012:cluster/dev-cluster"
	testArn3 := "arn:aws:ecs:us-east-1:123456789012:cluster/test-service"
	now := time.Now()

	mock := mockedEC2Cluster{
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
			names, err := listECSClusters(ctx, mock, resource.Scope{}, tc.configObj)
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}
