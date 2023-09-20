package resources

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

type mockedEC2Cluster struct {
	ecsiface.ECSAPI
	ListClustersOutput        ecs.ListClustersOutput
	DescribeClustersOutput    ecs.DescribeClustersOutput
	TagResourceOutput         ecs.TagResourceOutput
	ListTagsForResourceOutput ecs.ListTagsForResourceOutput
	DeleteClusterOutput       ecs.DeleteClusterOutput
}

func (m mockedEC2Cluster) ListClusters(*ecs.ListClustersInput) (*ecs.ListClustersOutput, error) {
	return &m.ListClustersOutput, nil
}

func (m mockedEC2Cluster) DescribeClusters(*ecs.DescribeClustersInput) (*ecs.DescribeClustersOutput, error) {
	return &m.DescribeClustersOutput, nil
}

func (m mockedEC2Cluster) TagResource(*ecs.TagResourceInput) (*ecs.TagResourceOutput, error) {
	return &m.TagResourceOutput, nil
}

func (m mockedEC2Cluster) ListTagsForResource(*ecs.ListTagsForResourceInput) (*ecs.ListTagsForResourceOutput, error) {
	return &m.ListTagsForResourceOutput, nil
}

func (m mockedEC2Cluster) DeleteCluster(*ecs.DeleteClusterInput) (*ecs.DeleteClusterOutput, error) {
	return &m.DeleteClusterOutput, nil
}

func TestEC2Cluster_GetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

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
						Value: aws.String(util.FormatTimestampTag(now)),
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
			expected:  []string{testArn1, testArn2},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile(testName1),
					}}},
			},
			expected: []string{testArn2},
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
			names, err := ec.getAll(context.Background(), config.Config{
				ECSCluster: tc.configObj,
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.StringValueSlice(names))
		})
	}

}

func TestEC2Cluster_NukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	ec := ECSClusters{
		Client: mockedEC2Cluster{
			DeleteClusterOutput: ecs.DeleteClusterOutput{},
		},
	}

	err := ec.nukeAll([]*string{aws.String("arn:aws:ecs:us-east-1:123456789012:cluster/cluster1")})
	require.NoError(t, err)
}
