package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// LoadBalancersV2 - represents all load balancers
type LoadBalancersV2 struct {
	Arns []string
}

// ResourceName - the simple name of the aws resource
func (balancer LoadBalancersV2) ResourceName() string {
	return "elbv2"
}

func (balancer LoadBalancersV2) MaxBatchSize() int {
	return 200
}

// ResourceIdentifiers - The arns of the load balancers
func (balancer LoadBalancersV2) ResourceIdentifiers() []string {
	return balancer.Arns
}

// Nuke - nuke 'em all!!!
func (balancer LoadBalancersV2) Nuke(session *session.Session, identifiers []string) error {
	if err := nukeAllElbv2Instances(session, awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
