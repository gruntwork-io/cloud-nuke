package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// EKSClusters - Represents all EKS clusters found in a region
type EKSClusters struct {
	Clusters []string
}

// ResourceName - The simple name of the aws resource
func (clusters EKSClusters) ResourceName() string {
	return "ekscluster"
}

// ResourceIdentifiers - The Name of the collected EKS clusters
func (clusters EKSClusters) ResourceIdentifiers() []string {
	return clusters.Clusters
}

func (clusters EKSClusters) MaxBatchSize() int {
	return 200
}

// Nuke - nuke all EKS Cluster resources
func (clusters EKSClusters) Nuke(awsSession *session.Session, identifiers []string) error {
	if err := nukeAllEksClusters(awsSession, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
