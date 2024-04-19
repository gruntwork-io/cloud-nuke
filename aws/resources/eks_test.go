package resources

import (
	"context"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/stretchr/testify/require"
)

type mockedEKSCluster struct {
	eksiface.EKSAPI

	ListClustersOutput    eks.ListClustersOutput
	DescribeClusterOutput eks.DescribeClusterOutput
	DeleteClusterOutput   eks.DeleteClusterOutput

	ListFargateProfilesOutput  eks.ListFargateProfilesOutput
	DeleteFargateProfileOutput eks.DeleteFargateProfileOutput

	DescribeNodegroupOutput eks.DescribeNodegroupOutput
	ListNodegroupsOutput    eks.ListNodegroupsOutput
	DeleteNodegroupOutput   eks.DeleteNodegroupOutput
}

func (m mockedEKSCluster) ListClusters(*eks.ListClustersInput) (*eks.ListClustersOutput, error) {
	// Only need to return mocked response output
	return &m.ListClustersOutput, nil
}

func (m mockedEKSCluster) DescribeCluster(*eks.DescribeClusterInput) (*eks.DescribeClusterOutput, error) {
	// Only need to return mocked response output
	return &m.DescribeClusterOutput, nil
}

func (m mockedEKSCluster) ListNodegroupsPages(
	input *eks.ListNodegroupsInput, fn func(*eks.ListNodegroupsOutput, bool) bool) error {
	// Only need to return mocked response output
	fn(&m.ListNodegroupsOutput, true)
	return nil
}

func (m mockedEKSCluster) DeleteNodegroup(*eks.DeleteNodegroupInput) (*eks.DeleteNodegroupOutput, error) {
	// Only need to return mocked response output
	return &m.DeleteNodegroupOutput, nil
}

func (m mockedEKSCluster) WaitUntilNodegroupDeleted(input *eks.DescribeNodegroupInput) error {
	return nil
}

func (m mockedEKSCluster) ListFargateProfilesPages(
	input *eks.ListFargateProfilesInput, fn func(*eks.ListFargateProfilesOutput, bool) bool) error {
	// Only need to return mocked response output
	fn(&m.ListFargateProfilesOutput, true)
	return nil
}

func (m mockedEKSCluster) DeleteFargateProfile(input *eks.DeleteFargateProfileInput) (*eks.DeleteFargateProfileOutput, error) {
	// Only need to return mocked response output
	return &m.DeleteFargateProfileOutput, nil
}

func (m mockedEKSCluster) WaitUntilFargateProfileDeleted(input *eks.DescribeFargateProfileInput) error {
	return nil
}

func (m mockedEKSCluster) WaitUntilClusterDeleted(input *eks.DescribeClusterInput) error {
	return nil
}

func (m mockedEKSCluster) DeleteCluster(input *eks.DeleteClusterInput) (*eks.DeleteClusterOutput, error) {
	return &m.DeleteClusterOutput, nil
}

func TestEKSClusterGetAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testClusterName := "test_cluster1"
	now := time.Now()
	eksCluster := EKSClusters{
		Client: mockedEKSCluster{
			ListClustersOutput: eks.ListClustersOutput{
				Clusters: []*string{awsgo.String(testClusterName)},
			},
			DescribeClusterOutput: eks.DescribeClusterOutput{
				Cluster: &eks.Cluster{CreatedAt: &now},
			},
		},
	}

	clusters, err := eksCluster.getAll(context.Background(), config.Config{})
	require.NoError(t, err)
	require.Contains(t, awsgo.StringValueSlice(clusters), testClusterName)
}

func TestEKSClusterNukeAll(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "")
	t.Parallel()

	testClusterName := "test_cluster1"
	testFargateProfileName := "test_fargate_profile1"
	testNodeGroupName := "test_nodegroup1"
	eksCluster := EKSClusters{
		Client: mockedEKSCluster{
			ListNodegroupsOutput: eks.ListNodegroupsOutput{
				Nodegroups: []*string{&testNodeGroupName},
			},
			DescribeClusterOutput: eks.DescribeClusterOutput{},
			ListFargateProfilesOutput: eks.ListFargateProfilesOutput{
				FargateProfileNames: []*string{&testFargateProfileName},
			},
			DeleteFargateProfileOutput: eks.DeleteFargateProfileOutput{},
			DeleteClusterOutput:        eks.DeleteClusterOutput{},
		},
	}

	err := eksCluster.nukeAll([]*string{&testClusterName})
	require.NoError(t, err)
}
