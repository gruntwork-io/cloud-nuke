package resources

import (
	"context"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/go-commons/errors"
)

// EgressOnlyInternetGateway represents all Egress only internet gateway
type EgressOnlyInternetGateway struct {
	BaseAwsResource
	Client ec2iface.EC2API
	Region string
	Pools  []string
}

func (egigw *EgressOnlyInternetGateway) Init(session *session.Session) {
	egigw.BaseAwsResource.Init(session)
	egigw.Client = ec2.New(session)
}

// ResourceName - the simple name of the aws resource
func (egigw *EgressOnlyInternetGateway) ResourceName() string {
	return "egress-only-internet-gateway"
}

func (egigw *EgressOnlyInternetGateway) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 49
}

// ResourceIdentifiers - The ids of the Egress only igw
func (egigw *EgressOnlyInternetGateway) ResourceIdentifiers() []string {
	return egigw.Pools
}

func (egigw *EgressOnlyInternetGateway) GetAndSetIdentifiers(c context.Context, configObj config.Config) ([]string, error) {
	identifiers, err := egigw.getAll(c, configObj)
	if err != nil {
		return nil, err
	}

	egigw.Pools = awsgo.StringValueSlice(identifiers)
	return egigw.Pools, nil
}

// Nuke - nuke 'em all!!!
func (egigw *EgressOnlyInternetGateway) Nuke(identifiers []string) error {
	if err := egigw.nukeAll(awsgo.StringSlice(identifiers)); err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}
