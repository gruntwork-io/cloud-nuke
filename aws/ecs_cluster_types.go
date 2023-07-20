package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// The pattern we use for running the `cloud-nuke` tool is to split the AWS API calls
// into batches when the function `NukeAllResources` is executed.
// A batch max number has been chosen for most modules.
// However, for ECS clusters there is no explicit limiting described in the AWS CLI docs.
// Therefore this `maxBatchSize` here is set to 49 as a safe maximum.
const maxBatchSize = 49

// ECSCluster - Represents all ECS clusters found in a region
type ECSCluster struct {
	Client      ecsiface.ECSAPI
	Region      string
	ClusterArns []string
}

// ResourceName - The simple name of the aws resource
func (clusters ECSCluster) ResourceName() string {
	return "ecs-cluster"
}

// ResourceIdentifiers - the collected ECS clusters
func (clusters ECSCluster) ResourceIdentifiers() []string {
	return clusters.ClusterArns
}

func (clusters ECSCluster) MaxBatchSize() int {
	return maxBatchSize
}

// Nuke - nuke all ECS Cluster resources
func (clusters ECSCluster) Nuke(awsSession *session.Session, identifiers []string) error {
	if err := nukeEcsClusters(awsSession, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
