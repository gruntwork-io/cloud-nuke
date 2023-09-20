package resources

import (
	"context"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// EKSClusters - Represents all EKS clusters found in a region
type EKSClusters struct {
	Client   eksiface.EKSAPI
	Region   string
	Clusters []string
}

func (clusters *EKSClusters) Init(session *session.Session) {
	clusters.Client = eks.New(session)
}

// ResourceName - The simple name of the aws resource
func (clusters *EKSClusters) ResourceName() string {
	return "ekscluster"
}

// ResourceIdentifiers - The Name of the collected EKS clusters
func (clusters *EKSClusters) ResourceIdentifiers() []string {
	return clusters.Clusters
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

	clusters.Clusters = awsgo.StringValueSlice(identifiers)
	return clusters.Clusters, nil
}

// Nuke - nuke all EKS Cluster resources
func (clusters *EKSClusters) Nuke(identifiers []string) error {
	if err := clusters.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}
