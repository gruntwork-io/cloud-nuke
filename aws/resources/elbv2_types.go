package resources

import (
	"context"

	"github.com/andrewderr/cloud-nuke-a1/config"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
	"github.com/gruntwork-io/go-commons/errors"
)

// LoadBalancersV2 - represents all load balancers
type LoadBalancersV2 struct {
	Client elbv2iface.ELBV2API
	Region string
	Arns   []string
}

func (balancer *LoadBalancersV2) Init(session *session.Session) {
	balancer.Client = elbv2.New(session)
}

// ResourceName - the simple name of the aws resource
func (balancer *LoadBalancersV2) ResourceName() string {
	return "elbv2"
}

func (balancer *LoadBalancersV2) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The arns of the load balancers
func (balancer *LoadBalancersV2) ResourceIdentifiers() []string {
	return balancer.Arns
}

func (balancer *LoadBalancersV2) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := balancer.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	balancer.Arns = awsgo.StringValueSlice(identifiers)
	return balancer.Arns, nil
}

// Nuke - nuke 'em all!!!
func (balancer *LoadBalancersV2) Nuke(identifiers []string) error {
	if err := balancer.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
