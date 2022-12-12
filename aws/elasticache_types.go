package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/go-commons/errors"
)

// Elasticaches - represents all Elasticache clusters
type Elasticaches struct {
	ClusterIds []string
}

// ResourceName - the simple name of the aws resource
func (cache Elasticaches) ResourceName() string {
	return "elasticache"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (cache Elasticaches) ResourceIdentifiers() []string {
	return cache.ClusterIds
}

func (cache Elasticaches) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (cache Elasticaches) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllElasticacheClusters(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
