package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb/elbiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// ELBv1 - represents all load balancers
type ELBv1 struct {
	Client elbiface.ELBAPI
	Region string
	Names  []string
}

// ResourceName - the simple name of the aws resource
func (balancer ELBv1) ResourceName() string {
	return "elbv1"
}

// ResourceIdentifiers - The names of the load balancers
func (balancer ELBv1) ResourceIdentifiers() []string {
	return balancer.Names
}

func (balancer ELBv1) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (balancer ELBv1) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllElbInstances(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type ElbDeleteError struct{}

func (e ElbDeleteError) Error() string {
	return "ELB was not deleted"
}
