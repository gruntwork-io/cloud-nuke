package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type ECSClustersAPI interface {
	DescribeClusters(ctx context.Context, params *ecs.DescribeClustersInput, optFns ...func(*ecs.Options)) (*ecs.DescribeClustersOutput, error)
	DeleteCluster(ctx context.Context, params *ecs.DeleteClusterInput, optFns ...func(*ecs.Options)) (*ecs.DeleteClusterOutput, error)
	ListClusters(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error)
	ListTagsForResource(ctx context.Context, params *ecs.ListTagsForResourceInput, optFns ...func(*ecs.Options)) (*ecs.ListTagsForResourceOutput, error)
	ListTasks(ctx context.Context, params *ecs.ListTasksInput, optFns ...func(*ecs.Options)) (*ecs.ListTasksOutput, error)
	StopTask(ctx context.Context, params *ecs.StopTaskInput, optFns ...func(*ecs.Options)) (*ecs.StopTaskOutput, error)
	TagResource(ctx context.Context, params *ecs.TagResourceInput, optFns ...func(*ecs.Options)) (*ecs.TagResourceOutput, error)
}

// ECSClusters - Represents all ECS clusters found in a region
type ECSClusters struct {
	BaseAwsResource
	Client      ECSClustersAPI
	Region      string
	ClusterArns []string
}

func (clusters *ECSClusters) InitV2(cfg aws.Config) {
	clusters.Client = ecs.NewFromConfig(cfg)
}

func (clusters *ECSClusters) IsUsingV2() bool { return true }

// ResourceName - The simple name of the aws resource
func (clusters *ECSClusters) ResourceName() string {
	return "ecscluster"
}

// ResourceIdentifiers - the collected ECS clusters
func (clusters *ECSClusters) ResourceIdentifiers() []string {
	return clusters.ClusterArns
}

func (clusters *ECSClusters) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.ECSCluster
}

func (clusters *ECSClusters) MaxBatchSize() int {
	return maxBatchSize
}

func (clusters *ECSClusters) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := clusters.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	clusters.ClusterArns = aws.ToStringSlice(identifiers)
	return clusters.ClusterArns, nil
}

// Nuke - nuke all ECS Cluster resources
func (clusters *ECSClusters) Nuke(identifiers []string) error {
	if err := clusters.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
