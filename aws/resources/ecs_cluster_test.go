package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
)

type mockedEC2Cluster struct {
	ecsiface.ECSAPI
	ListClustersOutput        ecs.ListClustersOutput
	DescribeClustersOutput    ecs.DescribeClustersOutput
	TagResourceOutput         ecs.TagResourceOutput
	ListTagsForResourceOutput ecs.ListTagsForResourceOutput
	DeleteClusterOutput       ecs.DeleteClusterOutput
	ListTasksOutput           ecs.ListTasksOutput
	StopTaskOutput            ecs.StopTaskOutput
}

func (m mockedEC2Cluster) ListClustersWithContext(_ aws.Context, _ *ecs.ListClustersInput, _ ...request.Option) (*ecs.ListClustersOutput, error) {
	return &m.ListClustersOutput, nil
}

func (m mockedEC2Cluster) DescribeClustersWithContext(_ aws.Context, _ *ecs.DescribeClustersInput, _ ...request.Option) (*ecs.DescribeClustersOutput, error) {
	return &m.DescribeClustersOutput, nil
}

func (m mockedEC2Cluster) TagResource(*ecs.TagResourceInput) (*ecs.TagResourceOutput, error) {
	return &m.TagResourceOutput, nil
}

func (m mockedEC2Cluster) ListTagsForResource(*ecs.ListTagsForResourceInput) (*ecs.ListTagsForResourceOutput, error) {
	return &m.ListTagsForResourceOutput, nil
}

func (m mockedEC2Cluster) DeleteClusterWithContext(_ aws.Context, _ *ecs.DeleteClusterInput, _ ...request.Option) (*ecs.DeleteClusterOutput, error) {
	return &m.DeleteClusterOutput, nil
}

func (m mockedEC2Cluster) ListTasksWithContext(_ aws.Context, _ *ecs.ListTasksInput, _ ...request.Option) (*ecs.ListTasksOutput, error) {
	return &m.ListTasksOutput, nil
}
func (m mockedEC2Cluster) StopTaskWithContext(_ aws.Context, _ *ecs.StopTaskInput, _ ...request.Option) (*ecs.StopTaskOutput, error) {
	return &m.StopTaskOutput, nil
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
				ClusterArns: []*string{
					aws.String(testArn1),
					aws.String(testArn2),
				},
			},

			DescribeClustersOutput: ecs.DescribeClustersOutput{
				Clusters: []*ecs.Cluster{
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
				Tags: []*ecs.Tag{
					{
						Key:   aws.String(util.FirstSeenTagKey),
						Value: aws.String(util.FormatTimestamp(now)),
					},
				},
			},
			ListTasksOutput: ecs.ListTasksOutput{
				TaskArns: []*string{
					aws.String("task-arn-001"),
					aws.String("task-arn-002"),
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
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := ec.getAll(tc.ctx, config.Config{
				ECSCluster: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}

}

func TestEC2Cluster_NukeAll(t *testing.T) {

	t.Parallel()

	ec := ECSClusters{
		Client: mockedEC2Cluster{
			DeleteClusterOutput: ecs.DeleteClusterOutput{},
		},
	}

	err := ec.nukeAll([]*string{aws.String("arn:aws:ecs:us-east-1:123456789012:cluster/cluster1")})
	require.NoError(t, err)
}

func TestEC2ClusterWithTasks_NukeAll(t *testing.T) {

	t.Parallel()

	ec := ECSClusters{
		Client: mockedEC2Cluster{
			DeleteClusterOutput: ecs.DeleteClusterOutput{},
			ListTasksOutput: ecs.ListTasksOutput{
				TaskArns: []*string{
					aws.String("task-arn-001"),
					aws.String("task-arn-002"),
				},
			},
		},
	}

	err := ec.nukeAll([]*string{aws.String("arn:aws:ecs:us-east-1:123456789012:cluster/cluster1")})
	require.NoError(t, err)
}
