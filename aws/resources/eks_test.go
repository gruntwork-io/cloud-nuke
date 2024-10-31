package resources

import (
	"context"
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
	DescribeClusterOutput        eks.DescribeClusterOutput
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
	return &m.DescribeClusterOutput, nil
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

	testClusterName := "test_cluster1"
	now := time.Now()
	eksCluster := EKSClusters{
		Client: mockedEKSCluster{
			ListClustersOutput: eks.ListClustersOutput{
				Clusters: []string{testClusterName},
			},
			DescribeClusterOutput: eks.DescribeClusterOutput{
				Cluster: &types.Cluster{CreatedAt: &now},
			},
		},
	}

	clusters, err := eksCluster.getAll(context.Background(), config.Config{})
	require.NoError(t, err)
	require.Contains(t, aws.ToStringSlice(clusters), testClusterName)
}

func TestEKSClusterNukeAll(t *testing.T) {
	t.Parallel()
	testClusterName := "test_cluster1"
	eksCluster := EKSClusters{
		BaseAwsResource: BaseAwsResource{
			Context: context.Background(),
		},
		Client: mockedEKSCluster{
			ListNodegroupsOutput:         eks.ListNodegroupsOutput{},
			DescribeClusterOutput:        eks.DescribeClusterOutput{},
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
