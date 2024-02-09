package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elb/elbiface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// LoadBalancers - represents all load balancers
type LoadBalancers struct {
	BaseAwsResource
	Client elbiface.ELBAPI
	Region string
	Names  []string
}

func (balancer *LoadBalancers) Init(session *session.Session) {
	balancer.Client = elb.New(session)
}

// ResourceName - the simple name of the aws resource
func (balancer *LoadBalancers) ResourceName() string {
	return "elb"
}

// ResourceIdentifiers - The names of the load balancers
func (balancer *LoadBalancers) ResourceIdentifiers() []string {
	return balancer.Names
}

func (balancer *LoadBalancers) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

func (balancer *LoadBalancers) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := balancer.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	balancer.Names = awsgo.StringValueSlice(identifiers)
	return balancer.Names, nil
}

// Nuke - nuke 'em all!!!
func (balancer *LoadBalancers) Nuke(identifiers []string) error {
	if err := balancer.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

type ElbDeleteError struct{}

func (e ElbDeleteError) Error() string {
	return "ELB was not deleted"
}
