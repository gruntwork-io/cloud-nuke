package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

type MSKClusterAPI interface {
	ListClustersV2(ctx context.Context, params *kafka.ListClustersV2Input, optFns ...func(*kafka.Options)) (*kafka.ListClustersV2Output, error)
	DeleteCluster(ctx context.Context, params *kafka.DeleteClusterInput, optFns ...func(*kafka.Options)) (*kafka.DeleteClusterOutput, error)
}

// MSKCluster - represents all AWS Managed Streaming for Kafka clusters that should be deleted.
type MSKCluster struct {
	BaseAwsResource
	Client      MSKClusterAPI
	Region      string
	ClusterArns []string
}

func (m *MSKCluster) Init(cfg aws.Config) {
	m.Client = kafka.NewFromConfig(cfg)
}

// ResourceName - the simple name of the aws resource
func (m *MSKCluster) ResourceName() string {
	return "msk-cluster"
}

// ResourceIdentifiers - The instance ids of the AWS Managed Streaming for Kafka clusters
func (m *MSKCluster) ResourceIdentifiers() []string {
	return m.ClusterArns
}

func (m *MSKCluster) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that nat gateway does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

func (m *MSKCluster) GetAndSetResourceConfig(configObj config.Config) config.ResourceType {
	return configObj.MSKCluster
}

func (m *MSKCluster) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := m.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	m.ClusterArns = aws.ToStringSlice(identifiers)
	return m.ClusterArns, nil
}

// Nuke - nuke 'em all!!!
func (m *MSKCluster) Nuke(identifiers []string) error {
	if err := m.nukeAll(aws.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
