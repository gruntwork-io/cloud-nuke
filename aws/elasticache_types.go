package aws

import "github.com/aws/aws-sdk-go/aws/session"

// Elasticaches - represents all Elasticache clusters
type Elasticaches struct {
	Arns []string
}

// ResourceName - the simple name of the aws resource
func (cache Elasticaches) ResourceName() string {
	return "elasticache"
}

// ResourceIdentifiers - The instance ids of the ec2 instances
func (cache Elasticaches) ResourceIdentifiers() []string {
	return cache.Arns
}

func (cache Elasticaches) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 200
}

// Nuke - nuke 'em all!!!
func (cache Elasticaches) Nuke(session *session.Session, identifiers []string) error {
	return nil
}
