package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// ECSCLusters - Represents all ECS clusters found in a region
type ECSCLusters struct {
	Clusters []string
}

// ResourceName - The simple name of the aws resource
func (clusters ECSCLusters) ResourceName() string {
	return "ecscluster"
}

// ResourceIdentifiers - the collected ECS clusters
func (clusters ECSCLusters) ResourceIdentifiers() []string {
	return clusters.Clusters
}

// MaxBatchSize - the maximum number of ECS clusters for a single request
func (clusters ECSCLusters) MaxBatchSize() int {
	return 200
}

// Nuke - nuke all ECS Cluster resources
func (clusters ECSCLusters) Nuke(awsSession *session.Session, identifiers []string) error {
	if err := nukeEcsClusters(awsSession, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
