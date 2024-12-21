package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type EKSClustersAPI interface {
	DeleteCluster(ctx context.Context, params *eks.DeleteClusterInput, optFns ...func(*eks.Options)) (*eks.DeleteClusterOutput, error)
	DeleteFargateProfile(ctx context.Context, params *eks.DeleteFargateProfileInput, optFns ...func(*eks.Options)) (*eks.DeleteFargateProfileOutput, error)
	DeleteNodegroup(ctx context.Context, params *eks.DeleteNodegroupInput, optFns ...func(*eks.Options)) (*eks.DeleteNodegroupOutput, error)
	DescribeCluster(ctx context.Context, params *eks.DescribeClusterInput, optFns ...func(*eks.Options)) (*eks.DescribeClusterOutput, error)
	DescribeFargateProfile(ctx context.Context, params *eks.DescribeFargateProfileInput, optFns ...func(*eks.Options)) (*eks.DescribeFargateProfileOutput, error)
	DescribeNodegroup(ctx context.Context, params *eks.DescribeNodegroupInput, optFns ...func(*eks.Options)) (*eks.DescribeNodegroupOutput, error)
	ListClusters(ctx context.Context, params *eks.ListClustersInput, optFns ...func(*eks.Options)) (*eks.ListClustersOutput, error)
	ListFargateProfiles(ctx context.Context, params *eks.ListFargateProfilesInput, optFns ...func(*eks.Options)) (*eks.ListFargateProfilesOutput, error)
	ListNodegroups(ctx context.Context, params *eks.ListNodegroupsInput, optFns ...func(*eks.Options)) (*eks.ListNodegroupsOutput, error)
}

// EKSClusters - Represents all EKS clusters found in a region
type EKSClusters struct {
	BaseAwsResource
	Client   EKSClustersAPI
	Region   string
	Clusters []string
}

func (clusters *EKSClusters) InitV2(cfg aws.Config) {
	clusters.Client = eks.NewFromConfig(cfg)
}

// ResourceName - The simple name of the aws resource
func (clusters *EKSClusters) ResourceName() string {
	return "ekscluster"
}

// ResourceIdentifiers - The Name of the collected EKS clusters
func (clusters *EKSClusters) ResourceIdentifiers() []string {
	return clusters.Clusters
}

func (clusters *EKSClusters) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.EKSCluster
}

func (clusters *EKSClusters) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that deleting EKS clusters involves deleting many
	// associated sub resources in tight loops, and they happen in parallel in go routines. We conservatively pick 10
	// here, both to limit overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

func (clusters *EKSClusters) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := clusters.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	clusters.Clusters = aws.ToStringSlice(identifiers)
	return clusters.Clusters, nil
}

// Nuke - nuke all EKS Cluster resources
func (clusters *EKSClusters) Nuke(identifiers []string) error {
	if err := clusters.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
