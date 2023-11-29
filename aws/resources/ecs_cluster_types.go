package resources

import (
	"context"

	"github.com/andrewderr/cloud-nuke-a1/config"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// The pattern we use for running the `cloud-nuke` tool is to split the AWS API calls
// into batches when the function `NukeAllResources` is executed.
// A batch max number has been chosen for most modules.
// However, for ECS clusters there is no explicit limiting described in the AWS CLI docs.
// Therefore this `maxBatchSize` here is set to 49 as a safe maximum.
const maxBatchSize = 49

// ECSClusters - Represents all ECS clusters found in a region
type ECSClusters struct {
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
