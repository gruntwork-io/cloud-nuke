package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// ECSClusters - Represents all ECS clusters found in a region
type ECSClusters struct {
	Clusters []string
}

// ResourceName - The simple name of the aws resource
func (clusters ECSClusters) ResourceName() string {
	return "ecsclust"
}

// ResourceIdentifiers - The ARNs of the collected ECS clusters
func (clusters ECSClusters) ResourceIdentifiers() []string {
	return clusters.Clusters
}

func (clusters ECSClusters) MaxBatchSize() int {
	return 200
}

// Nuke - nuke all ECS service resources
func (clusters ECSClusters) Nuke(awsSession *session.Session, identifiers []string) error {
	if err := nukeAllEcsClusters(awsSession, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
