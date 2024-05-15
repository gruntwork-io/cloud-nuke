package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// ECSClusters - Represents all ECS clusters found in a region
type ECSClusters struct {
	BaseAwsResource
	Client      ecsiface.ECSAPI
	Region      string
	ClusterArns []string
}

func (clusters *ECSClusters) Init(session *session.Session) {
	clusters.Client = ecs.New(session)
}

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

	clusters.ClusterArns = awsgo.StringValueSlice(identifiers)
	return clusters.ClusterArns, nil
}

// Nuke - nuke all ECS Cluster resources
func (clusters *ECSClusters) Nuke(identifiers []string) error {
	if err := clusters.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
