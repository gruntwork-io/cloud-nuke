package resources

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/stretchr/testify/require"
)

type mockedEKSCluster struct {
	EKSClustersAPI
	DeleteClusterOutput          eks.DeleteClusterOutput
	DeleteFargateProfileOutput   eks.DeleteFargateProfileOutput
	DeleteNodegroupOutput        eks.DeleteNodegroupOutput
	DescribeClusterOutputByName  map[string]*eks.DescribeClusterOutput
	DescribeFargateProfileOutput eks.DescribeFargateProfileOutput
	DescribeNodegroupOutput      eks.DescribeNodegroupOutput
	ListClustersOutput           eks.ListClustersOutput
	ListFargateProfilesOutput    eks.ListFargateProfilesOutput
	ListNodegroupsOutput         eks.ListNodegroupsOutput
}

func (m mockedEKSCluster) DeleteCluster(ctx context.Context, params *eks.DeleteClusterInput, optFns ...func(*eks.Options)) (*eks.DeleteClusterOutput, error) {
	return &m.DeleteClusterOutput, nil
}

func (m mockedEKSCluster) DeleteFargateProfile(ctx context.Context, params *eks.DeleteFargateProfileInput, optFns ...func(*eks.Options)) (*eks.DeleteFargateProfileOutput, error) {
	return &m.DeleteFargateProfileOutput, nil
}

func (m mockedEKSCluster) DeleteNodegroup(ctx context.Context, params *eks.DeleteNodegroupInput, optFns ...func(*eks.Options)) (*eks.DeleteNodegroupOutput, error) {
	return &m.DeleteNodegroupOutput, nil
}

func (m mockedEKSCluster) DescribeCluster(ctx context.Context, params *eks.DescribeClusterInput, optFns ...func(*eks.Options)) (*eks.DescribeClusterOutput, error) {
	return m.DescribeClusterOutputByName[aws.ToString(params.Name)], nil
}

func (m mockedEKSCluster) DescribeFargateProfile(ctx context.Context, params *eks.DescribeFargateProfileInput, optFns ...func(*eks.Options)) (*eks.DescribeFargateProfileOutput, error) {
	return &m.DescribeFargateProfileOutput, nil
}

func (m mockedEKSCluster) DescribeNodegroup(ctx context.Context, params *eks.DescribeNodegroupInput, optFns ...func(*eks.Options)) (*eks.DescribeNodegroupOutput, error) {
	return &m.DescribeNodegroupOutput, nil
}

func (m mockedEKSCluster) ListClusters(ctx context.Context, params *eks.ListClustersInput, optFns ...func(*eks.Options)) (*eks.ListClustersOutput, error) {
	return &m.ListClustersOutput, nil
}

func (m mockedEKSCluster) ListFargateProfiles(ctx context.Context, params *eks.ListFargateProfilesInput, optFns ...func(*eks.Options)) (*eks.ListFargateProfilesOutput, error) {
	return &m.ListFargateProfilesOutput, nil
}

func (m mockedEKSCluster) ListNodegroups(ctx context.Context, params *eks.ListNodegroupsInput, optFns ...func(*eks.Options)) (*eks.ListNodegroupsOutput, error) {
	return &m.ListNodegroupsOutput, nil
}
func TestEKSClusterGetAll(t *testing.T) {
	t.Parallel()

	testClusterName1 := "test_cluster1"
	testClusterName2 := "test_cluster2"
	testClusterName3 := "test_cluster3"
	now := time.Now()
	eksCluster := EKSClusters{
		Client: mockedEKSCluster{
			ListClustersOutput: eks.ListClustersOutput{
				Clusters: []string{testClusterName1, testClusterName2, testClusterName3},
			},
			DescribeClusterOutputByName: map[string]*eks.DescribeClusterOutput{
				testClusterName1: {
					Cluster: &types.Cluster{
						Name:      aws.String(testClusterName1),
						CreatedAt: &now,
						Tags:      map[string]string{"foo": "bar"},
					},
				},
				testClusterName2: {
					Cluster: &types.Cluster{
						Name:      aws.String(testClusterName1),
						CreatedAt: &now,
						Tags:      map[string]string{"foz": "boz"},
					},
				},
				testClusterName3: {
					Cluster: &types.Cluster{
						Name:      aws.String(testClusterName3),
						CreatedAt: &now,
						Tags:      map[string]string{"faz": "baz"},
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
			expected:  []string{testClusterName1, testClusterName2, testClusterName3},
		},
		"nameExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					NamesRegExp: []config.Expression{{
						RE: *regexp.MustCompile("test_cluster[12]"),
					}}},
			},
			expected: []string{testClusterName3},
		},
		"tagInclusionFilter": {
			configObj: config.ResourceType{
				IncludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"foo": {RE: *regexp.MustCompile("bar")}},
				},
			},
			expected: []string{testClusterName1},
		},
		"tagExclusionFilter": {
			configObj: config.ResourceType{
				ExcludeRule: config.FilterRule{
					Tags: map[string]config.Expression{"foo": {RE: *regexp.MustCompile("bar")}},
				},
			},
			expected: []string{testClusterName2, testClusterName3},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			names, err := eksCluster.getAll(context.Background(), config.Config{
				EKSCluster: tc.configObj,
			})

			require.NoError(t, err)
			require.Equal(t, tc.expected, aws.ToStringSlice(names))
		})
	}
}

func TestEKSClusterNukeAll(t *testing.T) {
	t.Parallel()
	testClusterName := "test_cluster1"
	eksCluster := EKSClusters{
		Client: mockedEKSCluster{
			ListNodegroupsOutput: eks.ListNodegroupsOutput{},
			DescribeClusterOutputByName: map[string]*eks.DescribeClusterOutput{
				testClusterName: {
					Cluster: &types.Cluster{
						Name:      aws.String(testClusterName),
						CreatedAt: aws.Time(time.Now()),
					},
				},
			},
			ListFargateProfilesOutput:    eks.ListFargateProfilesOutput{},
			DescribeNodegroupOutput:      eks.DescribeNodegroupOutput{},
			DeleteFargateProfileOutput:   eks.DeleteFargateProfileOutput{},
			DeleteClusterOutput:          eks.DeleteClusterOutput{},
			DescribeFargateProfileOutput: eks.DescribeFargateProfileOutput{},
		},
	}

	err := eksCluster.nukeAll([]*string{&testClusterName})
	require.NoError(t, err)
}
