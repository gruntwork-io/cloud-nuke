package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// MSKCluster - represents all AWS Managed Streaming for Kafka clusters that should be deleted.
type MSKCluster struct {
	ClusterArns []string
}

// ResourceName - the simple name of the aws resource
func (msk MSKCluster) ResourceName() string {
	return "msk-cluster"
}

// ResourceIdentifiers - The instance ids of the AWS Managed Streaming for Kafka clusters
func (msk MSKCluster) ResourceIdentifiers() []string {
	return msk.ClusterArns
}

func (msk MSKCluster) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle. Note that nat gateway does not support bulk delete, so
	// we will be deleting this many in parallel using go routines. We conservatively pick 10 here, both to limit
	// overloading the runtime and to avoid AWS throttling with many API calls.
	return 10
}

// Nuke - nuke 'em all!!!
func (ngw MSKCluster) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllMSKClusters(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
