package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// ECSClusters - Represents all ECS clusters found in a region
type ECSClusters struct {
	ClusterArns []string
}

// ResourceName - The simple name of the aws resource
func (clusters ECSClusters) ResourceName() string {
	return "ecscluster"
}

// ResourceIdentifiers - the collected ECS clusters
func (clusters ECSClusters) ResourceIdentifiers() []string {
	return clusters.ClusterArns
}

// MaxBatchSize - the maximum number of ECS clusters for a single request
func (clusters ECSClusters) MaxBatchSize() int {
	return 200
}

// Nuke - nuke all ECS Cluster resources
func (clusters ECSClusters) Nuke(awsSession *session.Session, identifiers []string) error {
	if err := nukeEcsClusters(awsSession, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
