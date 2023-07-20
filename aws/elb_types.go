package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb/elbiface"
	"github.com/gruntwork-io/go-commons/errors"
)

// LoadBalancers - represents all load balancers
type LoadBalancers struct {
	Client elbiface.ELBAPI
	Region string
	Names  []string
}

// ResourceName - the simple name of the aws resource
func (balancer LoadBalancers) ResourceName() string {
	return "elbv1"
}

// ResourceIdentifiers - The names of the load balancers
func (balancer LoadBalancers) ResourceIdentifiers() []string {
	return balancer.Names
}

func (balancer LoadBalancers) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// Nuke - nuke 'em all!!!
func (balancer LoadBalancers) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllElbInstances(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type ElbDeleteError struct{}

func (e ElbDeleteError) Error() string {
	return "ELB was not deleted"
}
